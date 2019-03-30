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
	"github.com/imdario/mergo"
	"github.com/pkg/errors"
	"github.com/sugarkube/sugarkube/internal/pkg/constants"
	"github.com/sugarkube/sugarkube/internal/pkg/convert"
	"github.com/sugarkube/sugarkube/internal/pkg/installable"
	"github.com/sugarkube/sugarkube/internal/pkg/interfaces"
	"github.com/sugarkube/sugarkube/internal/pkg/log"
	"github.com/sugarkube/sugarkube/internal/pkg/structs"
	"github.com/sugarkube/sugarkube/internal/pkg/utils"
	"gopkg.in/yaml.v2"
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

// Return the parallelisation if set
func (m Manifest) Parallelisation() uint16 {
	return m.manifestFile.Options.Parallelisation
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
		kappDescriptorAsMap := convert.KappDescriptorWithListsToMap(kappDescriptor)

		// need to merge structs for kapp descriptors (in order of lowest to highest precedence):
		//   * the kapp's sugarkube.yaml file (if we've acquired the kapp - will be prepended to the list
		//     of descriptors when it's loaded)
		//   * values from the sugarkube-conf.yaml file (if any are specified for
		//     programs the kapp declares in its `requires` block) (todo)
		//   * defaults in manifest files
		//   * the kapp descriptor in manifest files
		//   * overrides in stack files for the kapp
		//   * command line values (todo)

		descriptors := []structs.KappDescriptorWithMaps{
			manfestDefaults,
			kappDescriptorAsMap,
		}

		installableObj, err := installable.New(manifestId, descriptors)
		if err != nil {
			return nil, errors.WithStack(err)
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

// Updates the kappDescriptor struct with any overrides specified in the manifest file
func applyOverrides(kappDescriptor *structs.KappDescriptorWithLists, overrides map[string]interface{}) error {
	// todo - just create a map where sources are keyed on ID, then merge

	// we can't just unmarshal it to YAML, merge the overrides and marshal it again because overrides
	// use keys whose values are IDs of e.g. sources instead of referring to sources by index.
	overriddenState, ok := overrides[constants.StateKey]
	if ok {
		kappDescriptor.State = overriddenState.(string)
	}

	// update any overridden variables
	overriddenVars, ok := overrides[constants.VarsKey]
	if ok {
		overriddenVarsConverted, err := convert.MapInterfaceInterfaceToMapStringInterface(
			overriddenVars.(map[interface{}]interface{}))
		if err != nil {
			return errors.WithStack(err)
		}

		err = mergo.Merge(&kappDescriptor.Vars, overriddenVarsConverted, mergo.WithOverride)
		if err != nil {
			return errors.WithStack(err)
		}
	}

	// update sources
	overriddenSources, ok := overrides[constants.SourcesKey]
	if ok {
		overriddenSourcesConverted, err := convert.MapInterfaceInterfaceToMapStringInterface(
			overriddenSources.(map[interface{}]interface{}))
		if err != nil {
			return errors.WithStack(err)
		}

		currentAcquirers := kappDescriptor.Sources

		// sources are overridden by specifying the ID of a source as the key. So we need to iterate through
		// the overrides and also through the list of sources to update values
		for sourceId, v := range overriddenSourcesConverted {
			sourceOverridesMap, err := convert.MapInterfaceInterfaceToMapStringInterface(
				v.(map[interface{}]interface{}))
			if err != nil {
				return errors.WithStack(err)
			}

			for i, source := range kappDescriptor.Sources {
				if sourceId == currentAcquirers[i].Id {
					sourceYaml, err := yaml.Marshal(source)
					if err != nil {
						return errors.WithStack(err)
					}

					sourceMapInterface := map[string]interface{}{}
					err = yaml.Unmarshal(sourceYaml, sourceMapInterface)
					if err != nil {
						return errors.WithStack(err)
					}

					// we now have the overridden source values and the original source values as
					// types compatible for merging

					err = mergo.Merge(&sourceMapInterface, sourceOverridesMap, mergo.WithOverride)
					if err != nil {
						return errors.WithStack(err)
					}

					// convert the merged generic values back to a Source
					mergedSourceYaml, err := yaml.Marshal(sourceMapInterface)
					if err != nil {
						return errors.WithStack(err)
					}

					mergedSource := structs.Source{}
					err = yaml.Unmarshal(mergedSourceYaml, &mergedSource)
					if err != nil {
						return errors.WithStack(err)
					}

					log.Logger.Tracef("Updating source at index %d to: %#v", i, mergedSource)

					kappDescriptor.Sources[i] = mergedSource
				}
			}
		}
	}

	return nil
}

// Return overrides specified in the manifest associated with this kapp if there are any
func installableOverrides(manifestOverrides map[string]interface{}, installableId string) (map[string]interface{}, error) {
	rawOverrides, ok := manifestOverrides[installableId]
	if ok {
		overrides, err := convert.MapInterfaceInterfaceToMapStringInterface(
			rawOverrides.(map[interface{}]interface{}))
		if err != nil {
			return nil, errors.WithStack(err)
		}

		return overrides, nil
	}

	return nil, nil
}

// Load a single manifest file and parse the kapps it defines
func parseManifestFile(manifestFilePath string, manifestDescriptor structs.ManifestDescriptor) (interfaces.IManifest, error) {

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
// or it'll break creating a cache
func ValidateManifest(manifest interfaces.IManifest) error {
	ids := map[string]bool{}

	for _, kapp := range manifest.Installables() {
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
func SelectInstallables(manifests []interfaces.IManifest, includeSelector []string,
	excludeSelector []string) ([]interfaces.IInstallable, error) {
	var err error
	var match bool

	selectedKapps := make([]interfaces.IInstallable, 0)

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
					log.Logger.Debugf("Kapp '%s' matches selectors and "+
						"will be included in the results", installableObj.FullyQualifiedId())
					selectedKapps = append(selectedKapps, installableObj)
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
func MatchesSelector(installable interfaces.IInstallable, selector string) (bool, error) {

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
	}

	// todo - get rid of this once we've switched to an acquirer and can pull the path from a cache manager
	manifestDescriptor.Uri = uri

	manifestFilePath := uri

	// parse the manifest file we've acquired
	manifest, err := parseManifestFile(manifestFilePath, manifestDescriptor)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return manifest, nil
}
