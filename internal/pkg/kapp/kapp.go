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
	"bytes"
	"fmt"
	"github.com/imdario/mergo"
	"github.com/pkg/errors"
	"github.com/sugarkube/sugarkube/internal/pkg/acquirer"
	"github.com/sugarkube/sugarkube/internal/pkg/constants"
	"github.com/sugarkube/sugarkube/internal/pkg/convert"
	"github.com/sugarkube/sugarkube/internal/pkg/log"
	"github.com/sugarkube/sugarkube/internal/pkg/program"
	"github.com/sugarkube/sugarkube/internal/pkg/templater"
	"github.com/sugarkube/sugarkube/internal/pkg/utils"
	"github.com/sugarkube/sugarkube/internal/pkg/vars"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

type Template struct {
	Source string
	Dest   string
}

// Populated from the kapp's sugarkube.yaml file
type Config struct {
	program.RuntimeConfig
	Requires []string `yaml:"requires"`
}

type Kapp struct {
	Id          string
	manifest    *Manifest
	cacheDir    string
	Config      Config
	State       string
	Vars        map[string]interface{}
	PostActions []string `yaml:"post_actions"`
	Sources     []acquirer.Source
	Templates   []Template
}

const NamespaceSeparator = ":"

const PresentKey = "present"
const AbsentKey = "absent"

const StateKey = "state"
const SourcesKey = "sources"
const VarsKey = "vars"

// todo - allow templates to be overridden in manifest overrides blocks
//const TEMPLATES_KEY = "templates"

// Sets the root cache directory the kapp is checked out into
func (k *Kapp) SetCacheDir(cacheDir string) {
	log.Logger.Debugf("Setting cache dir on kapp '%s' to '%s'",
		k.FullyQualifiedId(), cacheDir)
	k.cacheDir = cacheDir
}

// Returns the fully-qualified ID of a kapp
func (k Kapp) FullyQualifiedId() string {
	if k.manifest == nil {
		return k.Id
	} else {
		return strings.Join([]string{k.manifest.Id(), k.Id}, NamespaceSeparator)
	}
}

// Return the manifest the kapp is in
func (k Kapp) GetManifest() *Manifest {
	return k.manifest
}

// Updates the kapp's struct after merging any manifest overrides
func (k *Kapp) refresh() error {
	manifestOverrides, err := k.manifestOverrides()
	if err != nil {
		return errors.WithStack(err)
	}

	if manifestOverrides != nil {
		// we can't just unmarshal it to YAML, merge the overrides and marshal it again because overrides
		// use keys whose values are IDs of e.g. sources instead of referring to sources by index.
		overriddenState, ok := manifestOverrides[StateKey]
		if ok {
			k.State = overriddenState.(string)
		}

		// update any overridden variables
		overriddenVars, ok := manifestOverrides[VarsKey]
		if ok {
			overriddenVarsConverted, err := convert.MapInterfaceInterfaceToMapStringInterface(
				overriddenVars.(map[interface{}]interface{}))
			if err != nil {
				return errors.WithStack(err)
			}

			err = mergo.Merge(&k.Vars, overriddenVarsConverted, mergo.WithOverride)
			if err != nil {
				return errors.WithStack(err)
			}
		}

		// update sources
		overriddenSources, ok := manifestOverrides[SourcesKey]
		if ok {
			overriddenSourcesConverted, err := convert.MapInterfaceInterfaceToMapStringInterface(
				overriddenSources.(map[interface{}]interface{}))
			if err != nil {
				return errors.WithStack(err)
			}

			currentAcquirers, err := k.Acquirers()
			if err != nil {
				return errors.WithStack(err)
			}

			// sources are overridden by specifying the ID of a source as the key. So we need to iterate through
			// the overrides and also through the list of sources to update values
			for sourceId, v := range overriddenSourcesConverted {
				sourceOverridesMap, err := convert.MapInterfaceInterfaceToMapStringInterface(
					v.(map[interface{}]interface{}))
				if err != nil {
					return errors.WithStack(err)
				}

				for i, source := range k.Sources {
					if sourceId == currentAcquirers[i].Id() {
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

						mergedSource := acquirer.Source{}
						err = yaml.Unmarshal(mergedSourceYaml, &mergedSource)
						if err != nil {
							return errors.WithStack(err)
						}

						log.Logger.Debugf("Updating kapp source at index %d to: %#v", i, mergedSource)

						k.Sources[i] = mergedSource
					}
				}
			}
		}
	}

	return nil
}

// Return overrides specified in the manifest associated with this kapp if there are any
func (k Kapp) manifestOverrides() (map[string]interface{}, error) {
	rawOverrides, ok := k.manifest.Overrides[k.Id]
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

// Returns the physical path to this kapp in a cache
func (k Kapp) CacheDir() string {
	cacheDir := filepath.Join(k.cacheDir, k.manifest.Id(), k.Id)

	// if no cache dir has been set (e.g. because the user is doing a dry-run),
	// don't return an absolute path
	if k.cacheDir != "" {
		absCacheDir, err := filepath.Abs(cacheDir)
		if err != nil {
			panic(fmt.Sprintf("Couldn't convert path to absolute path: %#v", err))
		}

		cacheDir = absCacheDir
	} else {
		log.Logger.Debug("No cache dir has been set on kapp. Cache dir will " +
			"not be converted to an absolute path.")
	}

	return cacheDir
}

// Returns certain kapp data that should be exposed as variables when running kapps
func (k Kapp) GetIntrinsicData() map[string]string {
	return map[string]string{
		"id":        k.Id,
		"state":     k.State,
		"cacheRoot": k.CacheDir(),
	}
}

// Returns an array of acquirers configured for the sources for this kapp. We need to recompute these each time
// instead of caching them so that any manifest overrides will take effect.
func (k *Kapp) Acquirers() ([]acquirer.Acquirer, error) {
	acquirers, err := acquirer.GetAcquirersFromSources(k.Sources)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return acquirers, nil
}

// Renders templates for the kapp and returns the paths they were written to
func (k *Kapp) RenderTemplates(mergedKappVars map[string]interface{}, stackConfig *StackConfig,
	dryRun bool) ([]string, error) {

	renderedPaths := make([]string, 0)

	if len(k.Templates) == 0 {
		log.Logger.Infof("No templates to render for kapp '%s'", k.FullyQualifiedId())
		return renderedPaths, nil
	}

	log.Logger.Infof("Rendering templates for kapp '%s'", k.FullyQualifiedId())

	for _, templateDefinition := range k.Templates {
		rawTemplateSource := templateDefinition.Source

		// run the source path through the templater in case it contains variables
		templateSource, err := templater.RenderTemplate(rawTemplateSource, mergedKappVars)
		if err != nil {
			return nil, errors.WithStack(err)
		}

		if !filepath.IsAbs(templateSource) {
			foundTemplate := false

			// see whether the template is in the kapp itself
			possibleSource := filepath.Join(k.CacheDir(), templateSource)
			log.Logger.Debugf("Searching for kapp template in '%s'", possibleSource)
			_, err := os.Stat(possibleSource)
			if err == nil {
				templateSource = possibleSource
				foundTemplate = true
			}

			if !foundTemplate {
				// search each template directory defined in the stack config
				for _, templateDir := range stackConfig.TemplateDirs {
					possibleSource := filepath.Join(stackConfig.Dir(), templateDir, templateSource)
					log.Logger.Debugf("Searching for kapp template in '%s'", possibleSource)
					_, err := os.Stat(possibleSource)
					if err == nil {
						templateSource = possibleSource
						foundTemplate = true
						break
					}
				}
			}

			if foundTemplate {
				log.Logger.Debugf("Found template at %s", templateSource)
			} else {
				return renderedPaths, errors.New(fmt.Sprintf("Failed to find template '%s' "+
					"in any of the defined template directories: %s", templateSource,
					strings.Join(stackConfig.TemplateDirs, ", ")))
			}
		}

		if !filepath.IsAbs(templateSource) {
			templateSource, err = filepath.Abs(templateSource)
			if err != nil {
				return renderedPaths, errors.WithStack(err)
			}
		}

		log.Logger.Debugf("Templating file '%s' with vars: %#v", templateSource, mergedKappVars)

		rawDestPath := templateDefinition.Dest
		// run the dest path through the templater in case it contains variables
		destPath, err := templater.RenderTemplate(rawDestPath, mergedKappVars)
		if err != nil {
			return nil, errors.WithStack(err)
		}

		if !filepath.IsAbs(destPath) {
			destPath = filepath.Join(k.CacheDir(), destPath)
		}

		// check whether the dest path exists
		if _, err := os.Stat(destPath); err == nil {
			log.Logger.Infof("Template destination path '%s' exists. "+
				"File will be overwritten by rendered template '%s' for kapp '%s'",
				destPath, templateSource, k.Id)
		}

		// check whether the parent directory for dest path exists and return an error if not
		destDir := filepath.Dir(destPath)
		if _, err := os.Stat(destDir); os.IsNotExist(err) {
			return renderedPaths, errors.New(fmt.Sprintf("Can't write template to non-existent directory: %s", destDir))
		}

		var outBuf bytes.Buffer

		err = templater.TemplateFile(templateSource, &outBuf, mergedKappVars)
		if err != nil {
			return renderedPaths, errors.WithStack(err)
		}

		if dryRun {
			log.Logger.Infof("Dry run. Template '%s' for kapp '%s' which "+
				"would be written to '%s' rendered as:\n%s", templateSource,
				k.Id, destPath, outBuf.String())
		} else {
			log.Logger.Infof("Writing rendered template '%s' for kapp "+
				"'%s' to '%s'", templateSource, k.FullyQualifiedId(), destPath)
			err := ioutil.WriteFile(destPath, outBuf.Bytes(), 0644)
			if err != nil {
				return renderedPaths, errors.WithStack(err)
			}

			renderedPaths = append(renderedPaths, destPath)
		}
	}

	return renderedPaths, nil
}

// Returns a boolean indicating whether the kapp matches the given selector
func (k Kapp) MatchesSelector(selector string) (bool, error) {

	selectorParts := strings.Split(selector, NamespaceSeparator)
	if len(selectorParts) != 2 {
		return false, errors.New(fmt.Sprintf("Fully-qualified kapps must be given, i.e. "+
			"formatted 'manifest-id%skapp-id' or 'manifest-id%s%s' for all kapps in a manifest",
			NamespaceSeparator, NamespaceSeparator, WildcardCharacter))
	}

	selectorManifestId := selectorParts[0]
	selectorKappId := selectorParts[1]

	kappIdParts := strings.Split(k.FullyQualifiedId(), NamespaceSeparator)
	if len(kappIdParts) != 2 {
		return false, errors.New(fmt.Sprintf("Fully-qualified kapp ID has an unexpected format: %s",
			k.FullyQualifiedId()))
	}

	kappManifestId := kappIdParts[0]
	kappId := kappIdParts[1]

	if selectorManifestId == kappManifestId {
		if selectorKappId == WildcardCharacter || selectorKappId == kappId {
			return true, nil
		}
	}

	return false, nil
}

// Finds all vars files for the given kapp and returns the result of merging
// all the data.
func (k *Kapp) getVarsFromFiles(stackConfig *StackConfig) (map[string]interface{}, error) {
	dirs, err := k.findVarsFiles(stackConfig)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	values := map[string]interface{}{}

	err = vars.Merge(&values, dirs...)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return values, nil
}

// This searches a directory tree from a given root path for files whose values
// should be merged together for a kapp. If a kapp instance is supplied, additional files
// will be searched for, in addition to stack-specific ones.
func (k *Kapp) findVarsFiles(stackConfig *StackConfig) ([]string, error) {
	precedence := []string{
		utils.StripExtension(constants.ValuesFile),
		stackConfig.Name,
		stackConfig.Provider,
		stackConfig.Provisioner,
		stackConfig.Account,
		stackConfig.Region,
		stackConfig.Profile,
		stackConfig.Cluster,
		constants.ProfileDir,
		constants.ClusterDir,
	}

	var kappId string

	// prepend the kapp ID to the precedence array
	precedence = append([]string{k.Id}, precedence...)

	acquirers, err := k.Acquirers()
	if err != nil {
		return nil, errors.WithStack(err)
	}

	for _, acquirerObj := range acquirers {
		precedence = append(precedence, acquirerObj.Id())

		id, err := acquirerObj.FullyQualifiedId()
		if err != nil {
			return nil, errors.WithStack(err)
		}

		precedence = append(precedence, id)
	}

	kappId = k.Id

	paths := make([]string, 0)

	for _, searchDir := range stackConfig.KappVarsDirs {
		searchPath, err := filepath.Abs(filepath.Join(stackConfig.Dir(), searchDir))
		if err != nil {
			return nil, errors.WithStack(err)
		}

		log.Logger.Infof("Searching for files/dirs for kapp '%s' under '%s' with basenames: %s",
			kappId, searchPath, strings.Join(precedence, ", "))

		err = utils.PrecedenceWalk(searchPath, precedence, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return errors.WithStack(err)
			}

			if !info.IsDir() {
				ext := filepath.Ext(path)

				if strings.ToLower(ext) != ".yaml" {
					log.Logger.Debugf("Ignoring non-yaml file: %s", path)
					return nil
				}

				log.Logger.Debugf("Adding kapp var file: %s", path)
				paths = append(paths, path)
			}

			return nil
		})

		if err != nil {
			return nil, errors.WithStack(err)
		}
	}

	log.Logger.Debugf("Kapp var paths for kapp '%s' are: %s", kappId,
		strings.Join(paths, ", "))

	return paths, nil
}
