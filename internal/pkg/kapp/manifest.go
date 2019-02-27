/*
 * Copyright 2018 The Sugarkube Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package kapp

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/sugarkube/sugarkube/internal/pkg/log"
	"github.com/sugarkube/sugarkube/internal/pkg/vars"
	"path/filepath"
	"strings"
)

type ManifestOptions struct {
	Parallelisation uint16
}

type Manifest struct {
	ConfiguredId  string `yaml:"id"` // a default will be used if no explicitly set. Used to namespace cache entries
	Uri           string
	UnparsedKapps []Kapp `yaml:"kapps"`
	kappsParsed   bool
	Overrides     map[string]interface{}
	Options       ManifestOptions
}

// Sets fields to default values
func (m *Manifest) Id() string {
	if len(m.ConfiguredId) > 0 {
		return m.ConfiguredId
	}

	// use the basename after stripping the extension by default
	// todo - get this from the acquirer for the manifest
	return strings.Replace(filepath.Base(m.Uri), filepath.Ext(m.Uri), "", 1)
}

// After parsing a YAML manifest, we need to add additional fields to each kapp. This method does so and
// returns the updated kapps. Having this method simplifies loading kapps because we can directly unmarshal
// them into a struct.
func (m *Manifest) ParsedKapps() []Kapp {
	if m.kappsParsed {
		return m.UnparsedKapps
	}

	// modify the unparsedKapps array since we won't need it in future - it's just a stepping stone after
	// loading the a manifest
	for i, unparsedKapp := range m.UnparsedKapps {
		unparsedKapp.manifest = m
		err := unparsedKapp.refresh()
		if err != nil {
			// todo - return this error after deciding how to deal with adding variables dynamically
			log.Logger.Fatalf("Error refreshing kapp: %#v - Error was: %#v", unparsedKapp, err)
		}
		m.UnparsedKapps[i] = unparsedKapp
	}

	m.kappsParsed = true

	return m.UnparsedKapps
}

// Load a single manifest file and parse the kapps it defines
// todo - change this to use an acquirer. Use the ID defined in the manifest
// settings YAML, or default to the manifest file basename.
func ParseManifestFile(path string) (*Manifest, error) {
	log.Logger.Infof("Parsing manifest: %s", path)

	data := map[string]interface{}{}
	err := vars.LoadYamlFile(path, &data)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	log.Logger.Debugf("Loaded manifest data: %#v", data)

	parsedManifest := Manifest{}
	err = vars.LoadYamlFile(path, &parsedManifest)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	parsedManifest.Uri = path

	log.Logger.Debugf("Returning manifest: %#v", parsedManifest)

	return &parsedManifest, nil
}

// Validates that there aren't multiple kapps with the same ID in the manifest,
// or it'll break creating a cache
func ValidateManifest(manifest *Manifest) error {
	ids := map[string]bool{}

	for _, kapp := range manifest.ParsedKapps() {
		id := kapp.Id

		if _, ok := ids[id]; ok {
			return errors.New(fmt.Sprintf("Multiple kapps exist with "+
				"the same id: %s", id))
		}

		acquirers, err := kapp.Acquirers()
		if err != nil {
			return errors.WithStack(err)
		}

		for _, acquirer := range acquirers {
			// verify all IDs can be generated successfully
			_, err := acquirer.FullyQualifiedId()
			if err != nil {
				return errors.WithStack(err)
			}
		}
	}

	return nil
}
