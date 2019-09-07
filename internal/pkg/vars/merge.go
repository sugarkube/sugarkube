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

package vars

import (
	"github.com/imdario/mergo"
	"github.com/pkg/errors"
	"github.com/sugarkube/sugarkube/internal/pkg/log"
	"gopkg.in/yaml.v2"
	"io/ioutil"
)

// Merges YAML files from multiple paths, with data from files loaded later
// overriding values loaded earlier.
func MergePaths(result *map[string]interface{}, paths ...string) error {

	for _, path := range paths {
		log.Logger.Debug("Loading path ", path)

		contents, err := ioutil.ReadFile(path)
		if err != nil {
			return errors.Wrapf(err, "Error reading file %s", path)
		}

		var yamlData = map[string]interface{}{}

		err = yaml.Unmarshal(contents, yamlData)
		if err != nil {
			return errors.Wrapf(err, "Error parsing YAML: %s", path)
		}

		log.Logger.Tracef("Merging %v with %v", result, yamlData)

		err = Merge(result, yamlData)
		if err != nil {
			return errors.WithStack(err)
		}
	}

	return nil
}

// Merges all given fragments, with values from later fragments overriding values
// from earlier ones.
func MergeFragments(result *map[string]interface{}, fragments ...map[string]interface{}) error {

	for _, fragment := range fragments {
		log.Logger.Tracef("Merging map %#v into existing map %#v - values "+
			"will be overridden", fragment, result)
		err := Merge(result, fragment)
		if err != nil {
			return errors.WithStack(err)
		}
	}

	return nil
}

// Merge two data structures overwriting lists
func Merge(dest interface{}, source interface{}) error {
	opts := make([]func(*mergo.Config), 0)

	// lists are always overridden - we don't append merged lists since it generally makes things more complicated
	opts = append(opts, mergo.WithOverride)

	err := mergo.Merge(dest, source, opts...)
	if err != nil {
		return errors.WithStack(err)
	}
	return nil
}
