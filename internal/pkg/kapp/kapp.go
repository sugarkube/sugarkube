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
	"github.com/pkg/errors"
	"github.com/sugarkube/sugarkube/internal/pkg/acquirer"
	"github.com/sugarkube/sugarkube/internal/pkg/convert"
	"github.com/sugarkube/sugarkube/internal/pkg/log"
	"gopkg.in/yaml.v2"
)

type installerConfig struct {
	kapp         string
	searchValues []string
	params       map[string]string
}

type Kapp struct {
	Id string
	// if true, this kapp should be present after completing, otherwise it
	// should be absent. This is here instead of e.g. putting all kapps into
	// an enclosing struct with 'present' and 'absent' properties so we can
	// preserve ordering. This approach lets users strictly define the ordering
	// of installation and deletion operations.
	ShouldBePresent bool
	installerConfig installerConfig
	Sources         []acquirer.Acquirer
	RootDir         string // root directory in a cache dir
}

const PRESENT_KEY = "present"
const ABSENT_KEY = "absent"
const SOURCES_KEY = "sources"

// Parses kapps and adds them to an array
func parseKapps(kapps *[]Kapp, kappDefinitions map[interface{}]interface{}, shouldBePresent bool) error {

	// parse each kapp definition
	for k, v := range kappDefinitions {
		kapp := Kapp{
			Id:              k.(string),
			ShouldBePresent: shouldBePresent,
		}

		log.Logger.Debugf("kapp=%#v, v=%#v", kapp, v)

		// parse the list of sources
		valuesMap, err := convert.MapInterfaceInterfaceToMapStringInterface(v.(map[interface{}]interface{}))
		if err != nil {
			return errors.Wrapf(err, "Error converting manifest value to map")
		}

		// marshal and unmarshal the list of sources
		sourcesBytes, err := yaml.Marshal(valuesMap[SOURCES_KEY])
		if err != nil {
			return errors.Wrapf(err, "Error marshalling sources yaml: %#v", v)
		}

		log.Logger.Debugf("Marshalled sources YAML: %s", sourcesBytes)

		sourcesMaps := []map[interface{}]interface{}{}
		err = yaml.UnmarshalStrict(sourcesBytes, &sourcesMaps)
		if err != nil {
			return errors.Wrapf(err, "Error unmarshalling yaml: %s", sourcesBytes)
		}

		log.Logger.Debugf("sourcesMaps=%#v", sourcesMaps)

		acquirers := make([]acquirer.Acquirer, 0)
		// now we have a list of sources, get the acquirer for each one
		for _, sourceMap := range sourcesMaps {
			sourceStringMap, err := convert.MapInterfaceInterfaceToMapStringString(sourceMap)
			if err != nil {
				return errors.WithStack(err)
			}

			acquirerImpl, err := acquirer.NewAcquirer(sourceStringMap)
			if err != nil {
				return errors.WithStack(err)
			}

			log.Logger.Debugf("Got acquirer %#v", acquirerImpl)

			acquirers = append(acquirers, acquirerImpl)
		}

		kapp.Sources = acquirers

		log.Logger.Debugf("Parsed kapp=%#v", kapp)

		*kapps = append(*kapps, kapp)
	}

	return nil
}

// Parses manifest YAML data and returns a list of kapps
func parseManifestYaml(data map[string]interface{}) ([]Kapp, error) {
	kapps := make([]Kapp, 0)

	presentKapps, ok := data[PRESENT_KEY]
	if ok {
		err := parseKapps(&kapps, presentKapps.(map[interface{}]interface{}), true)
		if err != nil {
			return nil, errors.Wrap(err, "Error parsing present kapps")
		}
	}

	absentKapps, ok := data[ABSENT_KEY]
	if ok {
		err := parseKapps(&kapps, absentKapps.(map[interface{}]interface{}), false)
		if err != nil {
			return nil, errors.Wrap(err, "Error parsing absent kapps")
		}
	}

	log.Logger.Debugf("Parsed kapps to install and remove: %#v", kapps)

	return kapps, nil
}
