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

type Manifest struct {
	// defaults to the file basename, but can be explicitly specified to avoid
	// clashes. This is also used to namespace entries in the cache.
	Id    string
	Uri   string
	Kapps []Kapp
}

func newManifest(uri string) Manifest {
	manifest := Manifest{
		Uri: uri,
	}

	SetManifestDefaults(&manifest)
	return manifest
}

// Sets fields to default values
func SetManifestDefaults(manifest *Manifest) {
	// use the basename after stripping the extension by default
	// todo - get this from the acquirer for the manifest
	defaultId := strings.Replace(filepath.Base(manifest.Uri), filepath.Ext(manifest.Uri), "", 1)

	if manifest.Id == "" {
		manifest.Id = defaultId
	}
}

// Load a single manifest file and parse the kapps it defines
// todo - change this to use an acquirer. Use the ID defined in the manifest
// settings YAML, or default to the manifest file basename.
func ParseManifestFile(path string) (*Manifest, error) {
	log.Logger.Debugf("Parsing manifest: %s", path)

	data, err := vars.LoadYamlFile(path)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	log.Logger.Debugf("Loaded manifest data: %#v", data)

	kapps, err := parseManifestYaml(data)

	manifest := newManifest(path)
	manifest.Kapps = kapps

	return &manifest, nil
}

// Parses manifest files and returns a list of manifests on success
// todo - this largely duplicates code in stack.LoadStackConfig. It needs
// centralising
//func ParseManifests(manifestPaths []string) ([]Manifest, error) {
//	log.Logger.Debugf("Parsing %d manifest(s)", len(manifestPaths))
//
//	manifests := make([]Manifest, 0)
//
//	for _, manifestPath := range manifestPaths {
//		manifest, err := ParseManifestFile(manifestPath)
//		if err != nil {
//			return nil, errors.WithStack(err)
//		}
//
//		manifests = append(manifests, *manifest)
//	}
//
//	return manifests, nil
//}

// Validates that there aren't multiple kapps with the same ID in the manifest,
// or it'll break creating a cache
func ValidateManifest(manifest *Manifest) error {
	ids := map[string]bool{}

	for _, kapp := range manifest.Kapps {
		id := kapp.Id

		if _, ok := ids[id]; ok {
			return errors.New(fmt.Sprintf("Multiple kapps exist with "+
				"the same id: %s", id))
		}

		for _, acquirer := range kapp.Sources {
			// verify all IDs can be generated successfully
			_, err := acquirer.Id()
			if err != nil {
				return errors.WithStack(err)
			}
		}
	}

	return nil
}
