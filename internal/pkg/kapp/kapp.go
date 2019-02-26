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
	"github.com/sugarkube/sugarkube/internal/pkg/acquirer"
	"github.com/sugarkube/sugarkube/internal/pkg/log"
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
	Id       string // todo - make private and add an accessor
	manifest *Manifest
	cacheDir string
	Config   Config
	State    string
	// todo - merge these values with the rest of the merged values prior to invoking a kapp
	Vars      map[string]interface{}
	Sources   []acquirer.Source
	Templates []Template
	acquirers []acquirer.Acquirer
}

const PRESENT_KEY = "present"
const ABSENT_KEY = "absent"
const SOURCES_KEY = "sources"
const TEMPLATES_KEY = "templates"
const VARS_KEY = "vars"
const ID_KEY = "id"
const STATE_KEY = "state"

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
		"State":     k.State,
		"cacheRoot": k.CacheDir(),
	}
}

// Returns an array of acquirers configured for the sources for this kapp
func (k *Kapp) Acquirers() ([]acquirer.Acquirer, error) {
	if len(k.acquirers) > 0 {
		return k.acquirers, nil
	}

	acquirers, err := acquirer.GetAcquirersFromSources(k.Sources)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	k.acquirers = acquirers

	return acquirers, nil
}

// Parses kapps and adds them to an array
//func parseKapps(manifest *Manifest, kapps *[]Kapp, kappDefinitions []interface{}, shouldBePresent bool) error {
//
//	// parse each kapp definition
//	for _, v := range kappDefinitions {
//		log.Logger.Debugf("Parsing kapp from values: %#v", v)
//
//		valuesMap, err := convert.MapInterfaceInterfaceToMapStringInterface(v.(map[interface{}]interface{}))
//		if err != nil {
//			return errors.Wrapf(err, "Error converting manifest value to map")
//		}
//
//		log.Logger.Debugf("kapp valuesMap=%#v", valuesMap)
//
//		// Return a useful error message if no ID has been declared. We need to do this here as well as when
//		// instantiating a kapp because this catches if the ID key is missing altogether
//		id, ok := valuesMap[ID_KEY].(string)
//		if !ok {
//			return errors.New(fmt.Sprintf("No ID declared for kapp: %#v", valuesMap))
//		}
//
//		state, ok := valuesMap[STATE_KEY].(string)
//		if !ok {
//			state = ""
//		}
//
//		vars, err := parseVariables(valuesMap)
//		if err != nil {
//			return errors.WithStack(err)
//		}
//
//		templates, err := parseTemplates(valuesMap)
//		if err != nil {
//			return errors.WithStack(err)
//		}
//
//		// prepare and parse the map containing acquirer values
//		sourcesBytes, err := yaml.Marshal(valuesMap[SOURCES_KEY])
//		if err != nil {
//			return errors.Wrapf(err, "Error marshalling sources yaml: %#v", valuesMap)
//		}
//
//		log.Logger.Debugf("Marshalled sources YAML: %s", sourcesBytes)
//
//		sourcesMaps := []map[interface{}]interface{}{}
//		err = yaml.UnmarshalStrict(sourcesBytes, &sourcesMaps)
//		if err != nil {
//			return errors.Wrapf(err, "Error unmarshalling yaml: %s", sourcesBytes)
//		}
//
//		log.Logger.Debugf("kapp sourcesMaps=%#v", sourcesMaps)
//
//		acquirers, err := acquirer.ParseAcquirers(sourcesMaps)
//		if err != nil {
//			return errors.WithStack(err)
//		}
//
//		kapp, err := newKapp(manifest, id, state, vars, templates, acquirers)
//
//		*kapps = append(*kapps, *kapp)
//	}
//
//	return nil
//}

// Parse variables in a Kapp values map
//func parseVariables(valuesMap map[string]interface{}) (map[string]interface{}, error) {
//	var kappVars map[string]interface{}
//
//	rawKappVars, ok := valuesMap[VARS_KEY]
//	if ok {
//		varsBytes, err := yaml.Marshal(rawKappVars)
//		if err != nil {
//			return nil, errors.Wrapf(err, "Error marshalling vars in kapp: %#v", valuesMap)
//		}
//
//		err = yaml.Unmarshal(varsBytes, &kappVars)
//		if err != nil {
//			return nil, errors.Wrapf(err, "Error unmarshalling vars for kapp: %#v", valuesMap)
//		}
//
//		log.Logger.Debugf("Parsed vars from kapp: %s", kappVars)
//	} else {
//		log.Logger.Debugf("No vars found in kapp")
//	}
//
//	return kappVars, nil
//}

// Parse templates from a Kapp values map
//func parseTemplates(valuesMap map[string]interface{}) ([]Template, error) {
//	templateBytes, err := yaml.Marshal(valuesMap[TEMPLATES_KEY])
//	if err != nil {
//		return nil, errors.Wrapf(err, "Error marshalling kapp templates: %#v", valuesMap)
//	}
//
//	log.Logger.Debugf("Marshalled templates YAML: %s", templateBytes)
//
//	templates := []Template{}
//	err = yaml.Unmarshal(templateBytes, &templates)
//	if err != nil {
//		return nil, errors.Wrapf(err, "Error unmarshalling template YAML: %s", templateBytes)
//	}
//
//	return templates, nil
//}

// Parses a manifest YAML. It separately parses all kapps that should be present and all those that should be
// absent, and returns a single list containing them all.
//func parseManifestYaml(manifest *Manifest, data map[string]interface{}) ([]Kapp, error) {
//	kapps := make([]Kapp, 0)
//
//	log.Logger.Debugf("Manifest data to parse: %#v", data)
//
//	presentKapps, ok := data[PRESENT_KEY]
//	if ok {
//		err := parseKapps(manifest, &kapps, presentKapps.([]interface{}), true)
//		if err != nil {
//			return nil, errors.Wrap(err, "Error parsing present kapps")
//		}
//	}
//
//	absentKapps, ok := data[ABSENT_KEY]
//	if ok {
//		err := parseKapps(manifest, &kapps, absentKapps.([]interface{}), false)
//		if err != nil {
//			return nil, errors.Wrap(err, "Error parsing absent kapps")
//		}
//	}
//
//	log.Logger.Debugf("Parsed kapps to install and remove: %#v", kapps)
//
//	return kapps, nil
//}
