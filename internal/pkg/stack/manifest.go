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
	"github.com/sugarkube/sugarkube/internal/pkg/acquirer"
	"github.com/sugarkube/sugarkube/internal/pkg/constants"
	"github.com/sugarkube/sugarkube/internal/pkg/convert"
	"github.com/sugarkube/sugarkube/internal/pkg/installable"
	"github.com/sugarkube/sugarkube/internal/pkg/interfaces"
	"github.com/sugarkube/sugarkube/internal/pkg/log"
	"github.com/sugarkube/sugarkube/internal/pkg/structs"
	"github.com/sugarkube/sugarkube/internal/pkg/utils"
	"path/filepath"
	"strings"
)

type Manifest struct {
	descriptor   structs.ManifestDescriptor
	manifestFile structs.ManifestFile
	installables []interfaces.IInstallable
}

// Sets fields to default values
func (m *Manifest) Id() string {
	if len(m.descriptor.Id) > 0 {
		return m.descriptor.Id
	}

	// use the basename after stripping the extension by default
	// todo - get this from the cache manager for the manifest
	return strings.Replace(filepath.Base(m.descriptor.Uri), filepath.Ext(m.descriptor.Uri), "", 1)
}

func (m *Manifest) Installables() []interfaces.IInstallable {
	return m.installables
}

// Return whether the manifest is sequential, i.e. whether each kapp in the manifest depends on the previous one
func (m Manifest) IsSequential() bool {
	return m.manifestFile.Options.IsSequential
}

// Instantiate installables for kapps defined in manifest files. Note: No overrides are applied at this stage.
func instantiateInstallables(manifestId string, manifest Manifest) ([]interfaces.IInstallable, error) {

	manifestFile := manifest.manifestFile

	installables := make([]interfaces.IInstallable, len(manifestFile.KappDescriptor))

	manfestDefaults := structs.KappDescriptorWithMaps{
		KappConfig: manifestFile.Defaults,
	}

	for i, kappDescriptor := range manifestFile.KappDescriptor {
		// convert the kappDescriptor to an installable
		kappDescriptorWithMap, err := convert.KappDescriptorWithListsToMap(kappDescriptor)
		if err != nil {
			return nil, errors.WithStack(err)
		}

		// need to merge structs for kapp descriptors (in order of lowest to highest precedence):
		//   * values from the sugarkube-conf.yaml file (if any are specified for
		//     programs the kapp declares in its `requires` block. This is a special case because we
		//     don't want to blindly merge all of these in, just the ones the kapp actually uses)
		//   * the kapp's sugarkube.yaml file (if we've acquired the kapp - will be prepended to the list
		//     of descriptors when it's loaded)
		//   * defaults in manifest files
		//   * the kapp descriptor in manifest files
		//   * overrides in stack files for the kapp
		//   * command line values (todo)

		descriptors := []structs.KappDescriptorWithMaps{
			manfestDefaults,
			kappDescriptorWithMap,
		}

		installableObj, err := installable.New(manifestId, descriptors)
		if err != nil {
			return nil, errors.WithStack(err)
		}

		// add descriptors for any versions declared
		for key, version := range manifest.descriptor.Versions {
			splitKey := strings.Split(key, constants.VersionSeparator)
			if splitKey[0] != installableObj.Id() {
				continue
			}

			descriptor := structs.KappDescriptorWithMaps{
				Sources: map[string]structs.Source{
					splitKey[1]: {
						Options: map[string]interface{}{
							// todo - use the right key depending on the acquirer (this
							//  currently assumes it's always a git acquirer)
							acquirer.BranchKey: version,
						},
					},
				},
			}

			err = installableObj.AddDescriptor(descriptor, false)
			if err != nil {
				return nil, errors.WithStack(err)
			}
		}

		// if there were any overrides defined in the stack for this installable, append
		// the descriptor to the list
		stackOverrides, ok := manifest.descriptor.Overrides[installableObj.Id()]
		if ok {
			err = installableObj.AddDescriptor(stackOverrides, false)
			if err != nil {
				return nil, errors.WithStack(err)
			}
		}

		installables[i] = installableObj
	}

	log.Logger.Tracef("Parsed installables from manifest '%s' as: %#v", manifestId, installables)

	return installables, nil
}

// Load a single manifest file and parse the kapps it defines
func ParseManifestFile(manifestFilePath string, manifestDescriptor structs.ManifestDescriptor) (interfaces.IManifest, error) {

	log.Logger.Infof("Parsing manifest file: %s", manifestFilePath)

	manifestFile := structs.ManifestFile{}

	err := utils.LoadYamlFile(manifestFilePath, &manifestFile)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	log.Logger.Tracef("Loaded raw manifest: %#v", manifestFile)

	manifest := Manifest{
		descriptor:   manifestDescriptor,
		manifestFile: manifestFile,
	}

	installables, err := instantiateInstallables(manifest.Id(), manifest)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	manifest.installables = installables

	return &manifest, nil
}

// Validates that there aren't multiple kapps with the same ID in the manifest,
// or it'll break creating a workspace
func ValidateManifest(manifest interfaces.IManifest) error {
	ids := map[string]bool{}

	reservedCharacters := []string{
		constants.RegistryFieldSeparator,
		constants.NamespaceSeparator,
		constants.WildcardCharacter,
	}

	for _, kapp := range manifest.Installables() {
		id := kapp.Id()

		if _, ok := ids[id]; ok {
			return errors.New(fmt.Sprintf("Multiple kapps exist with "+
				"the same id: %s", id))
		}

		// don't permit kapp IDs to contain reserved characters
		if strings.ContainsAny(id, strings.Join(reservedCharacters, "")) {
			return errors.New(fmt.Sprintf("Kapp IDs can't contain any of the reserved "+
				"characters: '%s'", strings.Join(reservedCharacters, "")))
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
func SelectInstallables(manifests []interfaces.IManifest, includeSelector []string,
	excludeSelector []string) ([]interfaces.IInstallable, error) {
	var err error
	var match bool

	selectedInstallables := make([]interfaces.IInstallable, 0)

	for _, manifest := range manifests {
		for _, installableObj := range manifest.Installables() {
			match = false
			// a kapp is selected either if it matches an include selector or there are no include selectors,
			// and it doesn't match an exclude selector
			for _, selector := range includeSelector {
				match, err = MatchesSelector(installableObj, selector)
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
					"exclude selectors", installableObj.FullyQualifiedId())

				match = false

				for _, selector := range excludeSelector {
					match, err = MatchesSelector(installableObj, selector)
					if err != nil {
						return nil, errors.WithStack(err)
					}

					if match {
						break
					}
				}

				if !match {
					log.Logger.Debugf("Kapp '%s' matches selector",
						installableObj.FullyQualifiedId())
					selectedInstallables = append(selectedInstallables, installableObj)
				}
			}
		}
	}

	log.Logger.Debugf("Selected %d kapps", len(selectedInstallables))
	for _, installableObj := range selectedInstallables {
		log.Logger.Debugf("Selected: %s", installableObj)
	}

	log.Logger.Infof("%d kapps selected for processing in total", len(selectedInstallables))

	// if nothing matched, return an error
	if len(selectedInstallables) == 0 {
		return nil, errors.New(fmt.Sprintf("Nothing was matched by including '%s' and "+
			"excluding '%s'", strings.Join(includeSelector, ", "),
			strings.Join(excludeSelector, ", ")))
	}

	return selectedInstallables, nil
}

// Returns a boolean indicating whether the installable matches the given selector
func MatchesSelector(installable interfaces.IInstallable, selector string) (bool, error) {

	log.Logger.Tracef("Testing whether installable '%s' matches the selector '%s'",
		installable.FullyQualifiedId(), selector)

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
			log.Logger.Tracef("Installable '%s' did match the selector '%s'", installable.FullyQualifiedId(), selector)
			return true, nil
		}
	}

	log.Logger.Tracef("Installable '%s' didn't match the selector '%s'", installable.FullyQualifiedId(), selector)
	return false, nil
}

func acquireManifests(stackObj structs.StackFile) ([]interfaces.IManifest, error) {
	log.Logger.Info("Acquiring manifests...")

	manifests := make([]interfaces.IManifest, len(stackObj.ManifestDescriptors))

	for i, manifestDescriptor := range stackObj.ManifestDescriptors {
		manifest, err := acquireManifest(filepath.Dir(stackObj.FilePath), manifestDescriptor)
		if err != nil {
			return nil, errors.WithStack(err)
		}

		manifests[i] = manifest
	}

	return manifests, nil
}

// Acquires a manifest.
// todo - refactor to use an acquirer
func acquireManifest(stackConfigFileDir string, manifestDescriptor structs.ManifestDescriptor) (interfaces.IManifest, error) {

	// The file acquirer needs to convert relative paths to absolute.
	uri := manifestDescriptor.Uri
	if !filepath.IsAbs(uri) {
		uri = filepath.Join(stackConfigFileDir, uri)
		log.Logger.Debugf("Fiddling manifest URI to '%s' (joined with stack file dir '%s')", uri, stackConfigFileDir)
	}

	// todo - get rid of this once we've switched to an acquirer and can pull the path from a cache manager
	manifestDescriptor.Uri = uri

	manifestFilePath := uri

	// parse the manifest file we've acquired
	manifest, err := ParseManifestFile(manifestFilePath, manifestDescriptor)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return manifest, nil
}
