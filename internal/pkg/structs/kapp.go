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

package structs

// Structs to load a kapp's sugarkube.yaml file

// A fragment of configuration for a program or kapp. It can be loaded either
// from a kapp's sugarkube.yaml file or the global sugarkube config file. It
// allows default env vars and arguments to be configured in one place and reused.
type ProgramConfig struct {
	EnvVars map[string]interface{}                    `yaml:"envVars"`
	Version string                                    `yaml:"version"`
	Args    map[string]map[string][]map[string]string `yaml:"args"`
}

// A struct for an actual sugarkube.yaml file
type KappConfig struct {
	ProgramConfig `yaml:",inline"`
	Requires      []string `yaml:"requires"`
	PostActions   []string `yaml:"post_actions"`
	Templates     []Template
	Vars          map[string]interface{}
	Sources       []Source
}
