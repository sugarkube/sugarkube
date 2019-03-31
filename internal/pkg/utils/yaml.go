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

package utils

import (
	"github.com/pkg/errors"
	"github.com/sugarkube/sugarkube/internal/pkg/log"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
	"path/filepath"
)

// Loads a YAML file
func LoadYamlFile(path string, out interface{}) error {
	// make sure the file exists
	absPath, err := filepath.Abs(path)
	if err != nil {
		return errors.WithStack(err)
	}

	if _, err := os.Stat(absPath); err != nil {
		log.Logger.Debugf("YAML file doesn't exist: %s", absPath)
		return errors.WithStack(err)
	}

	yamlData, err := ioutil.ReadFile(path)
	if err != nil {
		return errors.Wrapf(err, "Error reading YAML file %s", path)
	}

	err = yaml.Unmarshal(yamlData, out)
	if err != nil {
		return errors.Wrapf(err, "Error loading YAML file %s", path)
	}

	log.Logger.Tracef("Loaded YAML file: %#v", out)

	return nil
}

//// Returns a YAML representation of an object
//func AsYaml(in interface{}) (string, error) {
//	yamlData, err := yaml.Marshal(in)
//	if err != nil {
//		return "", errors.WithStack(err)
//	}
//
//	return string(yamlData[:]), nil
//}
