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
	"github.com/pkg/errors"
	"github.com/sugarkube/sugarkube/internal/pkg/log"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
	"path/filepath"
)

// Loads a YAML file
func LoadYamlFile(path string) (map[string]interface{}, error) {
	// make sure the file exists
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	if _, err := os.Stat(absPath); err != nil {
		log.Logger.Debugf("YAML file doesn't exist: %s", absPath)
		return nil, errors.WithStack(err)
	}

	yamlData, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, errors.Wrapf(err, "Error reading YAML file %s", path)
	}

	data := map[string]interface{}{}

	err = yaml.Unmarshal(yamlData, data)
	if err != nil {
		return nil, errors.Wrapf(err, "Error loading YAML file %s", path)
	}

	log.Logger.Debugf("YAML file: %#v", data)

	return data, nil
}
