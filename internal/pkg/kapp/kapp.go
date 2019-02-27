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
	"github.com/imdario/mergo"
	"github.com/pkg/errors"
	"github.com/sugarkube/sugarkube/internal/pkg/acquirer"
	"github.com/sugarkube/sugarkube/internal/pkg/convert"
	"github.com/sugarkube/sugarkube/internal/pkg/log"
	"gopkg.in/yaml.v2"
	"path/filepath"
	"strings"
)

type Template struct {
	Source string
	Dest   string
}

// Populated from the kapp's sugarkube.yaml file
type Config struct {
	EnvVars    map[string]interface{} `yaml:"envVars"`
	Version    string
	TargetArgs map[string]map[string][]map[string]string `yaml:"targets"`
}

type Kapp struct {
	Id        string
	manifest  *Manifest
	cacheDir  string
	Config    Config
	State     string
	Vars      map[string]interface{}
	Sources   []acquirer.Source
	Templates []Template
}

const PRESENT_KEY = "present"
const ABSENT_KEY = "absent"

const STATE_KEY = "state"
const SOURCES_KEY = "sources"
const VARS_KEY = "vars"

// todo - allow templates to be overridden in manifest overides blocks
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
		return strings.Join([]string{k.manifest.Id(), k.Id}, ":")
	}
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
		overriddenState, ok := manifestOverrides[STATE_KEY]
		if ok {
			k.State = overriddenState.(string)
		}

		// update any overridden variables
		overriddenVars, ok := manifestOverrides[VARS_KEY]
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
		overriddenSources, ok := manifestOverrides[SOURCES_KEY]
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
