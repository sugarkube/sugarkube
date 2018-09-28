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
	"github.com/sugarkube/sugarkube/internal/pkg/convert"
	"github.com/sugarkube/sugarkube/internal/pkg/log"
	"gopkg.in/yaml.v2"
)

type installerConfig struct {
	kapp         string
	searchValues []string
	params       map[string]string
}

type Template struct {
	Source string
	Dest   string
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
	Templates       []Template
	RootDir         string // root directory in a cache dir
}

const PRESENT_KEY = "present"
const ABSENT_KEY = "absent"
const SOURCES_KEY = "sources"
const TEMPLATES_KEY = "templates"
const ID_KEY = "id"

func (k *Kapp) AsMap() map[string]string {
	return map[string]string{
		"id":              k.Id,
		"shouldBePresent": fmt.Sprintf("%#v", k.ShouldBePresent),
		"rootDir":         k.RootDir,
	}
}

// Parses kapps and adds them to an array
func parseKapps(kapps *[]Kapp, kappDefinitions []interface{}, shouldBePresent bool) error {

	// parse each kapp definition
	for _, v := range kappDefinitions {
		kapp := Kapp{
			ShouldBePresent: shouldBePresent,
		}

		log.Logger.Debugf("kapp=%#v, v=%#v", kapp, v)

		// parse the list of sources
		valuesMap, err := convert.MapInterfaceInterfaceToMapStringInterface(v.(map[interface{}]interface{}))
		log.Logger.Debugf("valuesMap=%#v", valuesMap)

		kapp.Id = valuesMap[ID_KEY].(string)

		if err != nil {
			return errors.Wrapf(err, "Error converting manifest value to map")
		}

		// marshal and unmarshal the list of templates
		templateBytes, err := yaml.Marshal(valuesMap[TEMPLATES_KEY])
		if err != nil {
			return errors.Wrapf(err, "Error marshalling kapp templates: %#v", v)
		}

		log.Logger.Debugf("Marshalled templates YAML: %s", templateBytes)

		templates := []Template{}
		err = yaml.Unmarshal(templateBytes, &templates)
		if err != nil {
			return errors.Wrapf(err, "Error unmarshalling template YAML: %s", templateBytes)
		}
		kapp.Templates = templates

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

	log.Logger.Debugf("Manifest data to parse: %#v", data)

	presentKapps, ok := data[PRESENT_KEY]
	if ok {
		err := parseKapps(&kapps, presentKapps.([]interface{}), true)
		if err != nil {
			return nil, errors.Wrap(err, "Error parsing present kapps")
		}
	}

	absentKapps, ok := data[ABSENT_KEY]
	if ok {
		err := parseKapps(&kapps, absentKapps.([]interface{}), false)
		if err != nil {
			return nil, errors.Wrap(err, "Error parsing absent kapps")
		}
	}

	log.Logger.Debugf("Parsed kapps to install and remove: %#v", kapps)

	return kapps, nil
}
