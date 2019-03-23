/*
 * Copyright 2019 The Sugarkube Authors
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

package stack

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/sugarkube/sugarkube/internal/pkg/constants"
	"github.com/sugarkube/sugarkube/internal/pkg/installables"
	"github.com/sugarkube/sugarkube/internal/pkg/log"
	"github.com/sugarkube/sugarkube/internal/pkg/structs"
	"github.com/sugarkube/sugarkube/internal/pkg/vars"
	"path/filepath"
	"strings"
)

type Manifest struct {
	address      structs.ManifestAddress
	rawConfig    structs.Manifest
	installables []*installables.Installable
}

// Sets fields to default values
func (m *Manifest) Id() string {
	if len(m.address.Id) > 0 {
		return m.address.Id
	}

	// use the basename after stripping the extension by default
	// todo - get this from the acquirer for the manifest
	return strings.Replace(filepath.Base(m.address.Uri), filepath.Ext(m.address.Uri), "", 1)
}

// After parsing a YAML manifest, we need to add additional fields to each kapp. This method does so and
// returns the updated kapps. Having this method simplifies loading kapps because we can directly unmarshal
// them into a struct.
func (m *Manifest) ParsedKapps() []installables.Installable {
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
		id := kapp.Id()

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

// Return installables selected by inclusion/exclusion selectors from the given
// manifests. Installables will be returned in the order they appear in the manifests
// regardless of the orders of the selectors.
func SelectInstallables(manifests []*Manifest, includeSelector []string,
	excludeSelector []string) ([]installables.Installable, error) {
	var err error
	var match bool

	selectedKapps := make([]installables.Installable, 0)

	for _, manifest := range manifests {
		for _, installable := range manifest.ParsedKapps() {
			match = false
			// a kapp is selected either if it matches an include selector or there are no include selectors,
			// and it doesn't match an exclude selector
			for _, selector := range includeSelector {
				match, err = MatchesSelector(installable, selector)
				if err != nil {
					return nil, errors.WithStack(err)
				}

				if match {
					break
				}
			}

			// kapp is a possible candidate. Only add it to the result set if
			// it doesn't match any exclude selectors
			if len(includeSelector) == 0 || match {
				log.Logger.Debugf("Kapp '%s' is a candidate... testing "+
					"exclude selectors", installable.FullyQualifiedId())

				match = false

				for _, selector := range excludeSelector {
					match, err = MatchesSelector(installable, selector)
					if err != nil {
						return nil, errors.WithStack(err)
					}

					if match {
						break
					}
				}

				if !match {
					log.Logger.Debugf("Kapp '%s' matches selectors and "+
						"will be included in the results", installable.FullyQualifiedId())
					selectedKapps = append(selectedKapps, installable)
				}
			}
		}
	}

	log.Logger.Debugf("Selected %d kapps", len(selectedKapps))
	for _, selectedKapp := range selectedKapps {
		log.Logger.Debugf("Selected: %s", selectedKapp.FullyQualifiedId())
	}

	log.Logger.Infof("%d kapps selected for processing in total", len(selectedKapps))

	return selectedKapps, nil
}

// Returns a boolean indicating whether the installable matches the given selector
func MatchesSelector(installable installables.Installable, selector string) (bool, error) {

	selectorParts := strings.Split(selector, constants.NamespaceSeparator)
	if len(selectorParts) != 2 {
		return false, errors.New(fmt.Sprintf("Fully-qualified IDs must "+
			"be given, i.e. formatted 'manifest-id%skapp-id' or 'manifest-id%s%s' "+
			"for all kapps in a manifest", constants.NamespaceSeparator,
			constants.NamespaceSeparator, constants.WildcardCharacter))
	}

	selectorManifestId := selectorParts[0]
	selectorId := selectorParts[1]

	idParts := strings.Split(installable.FullyQualifiedId(), constants.NamespaceSeparator)
	if len(idParts) != 2 {
		return false, errors.New(fmt.Sprintf("Fully-qualified kapp ID "+
			"has an unexpected format: %s", installable.FullyQualifiedId()))
	}

	kappManifestId := idParts[0]
	kappId := idParts[1]

	if selectorManifestId == kappManifestId {
		if selectorId == constants.WildcardCharacter || selectorId == kappId {
			return true, nil
		}
	}

	return false, nil
}

// Acquires a manifest.
// todo - refactor to use an acquirer
func acquireManifest(stackConfigFileDir string, manifestAddress structs.ManifestAddress) (*Manifest, error) {

	// The file acquirer needs to convert relative paths to absolute.
	uri := manifestAddress.Uri
	if !filepath.IsAbs(uri) {
		uri = filepath.Join(stackConfigFileDir, uri)
	}

	// parse the manifests
	parsedManifest, err := ParseManifestFile(uri)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	// todo - remove this. It should be handled by an acquirer
	//SetManifestDefaults(&manifest)
	parsedManifest.ConfiguredId = manifest.ConfiguredId
	parsedManifest.Overrides = manifest.Overrides

	return parsedManifest, nil
}
