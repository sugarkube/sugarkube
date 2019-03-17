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

package program

import (
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

// A fragment of configuration for a program or kapp. It can be loaded either
// from a kapp's sugarkube.yaml file or the global sugarkube config file. It
// allows default env vars and arguments to be configured in one place and reused.
type Config struct {
	EnvVars map[string]interface{}                    `yaml:"envVars"`
	Version string                                    `yaml:"version"`
	Args    map[string]map[string][]map[string]string `yaml:"args"`
}

// Returns a YAML representation of the config
func (c Config) AsYaml() (string, error) {
	yamlData, err := yaml.Marshal(c)
	if err != nil {
		return "", errors.WithStack(err)
	}

	return string(yamlData[:]), nil
}
