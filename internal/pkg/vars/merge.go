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

func Merge(result *map[string]interface{}, paths ...string) error {

	for _, path := range paths {

		log.Logger.Debug("Loading path ", path)

		yamlFile, err := ioutil.ReadFile(path)
		if err != nil {
			return errors.Wrapf(err, "Error reading YAML file %s", path)
		}

		var loaded = map[string]interface{}{}

		err = yaml.Unmarshal(yamlFile, loaded)
		if err != nil {
			return errors.Wrapf(err, "Error loading YAML file: %s", path)
		}

		log.Logger.Debugf("Merging %v with %v", result, loaded)

		mergo.Merge(result, loaded, mergo.WithOverride)
	}

	return nil
}
