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

package config

import (
	"github.com/stretchr/testify/assert"
	"github.com/sugarkube/sugarkube/internal/pkg/log"
	"github.com/sugarkube/sugarkube/internal/pkg/structs"
	"os"
	"path"
	"testing"
)

const testDir = "../../testdata"

func init() {
	log.ConfigureLogger("debug", false, os.Stderr)
}

// Test that registry values override values when returning templated vars
func TestLoadConfig(t *testing.T) {
	ten := uint8(10)
	twenty := uint8(20)
	thirty := uint8(30)

	expectedConfig := &Config{
		JsonLogs:   false,
		LogLevel:   "warn",
		NumWorkers: 5,
		Programs: map[string]structs.KappConfig{
			"proga": {
				Vars: map[string]interface{}{
					"kubeconfig": "{{ .kubeconfig }}",
					"release":    "{{ .kapp.vars.release | default .kapp.id }}",
					"helm":       "/path/to/helm",
				},
				RunUnits: map[string]structs.RunUnit{
					"proga": {
						WorkingDir: "/tmp",
						EnvVars: map[string]string{
							"user": "sk",
						},
						PlanInstall: []structs.RunStep{
							{
								Name:    "print-hi",
								Command: "echo",
								Args:    "hi",
							},
						},
						ApplyInstall: []structs.RunStep{
							{
								Name:    "do-stuff-second",
								Command: "{{ .kapp.vars.helm }}",
								Args:    "do-stuff {{ .kapp.vars.release }}",
								EnvVars: map[string]string{
									"KUBECONFIG": "{{ .kapp.vars.kubeconfig }}",
								},
								MergePriority: &thirty,
							},
						},
					},
				},
			},
			"prog2": {
				Vars: map[string]interface{}{
					"kubeconfig": "{{ .kubeconfig }}",
					"region":     "{{ .stack.region }}",
				},
				RunUnits: map[string]structs.RunUnit{
					"prog2": {
						Binaries: []string{"cat"},
						ApplyInstall: []structs.RunStep{
							{
								Name:    "do-stuff-first",
								Command: "/path/to/prog2",
								Args:    "do-stuff-zzz {{ .kapp.vars.region }}",
								EnvVars: map[string]string{
									"REGION": "{{ .kapp.vars.region }}",
									"COLOUR": "blue",
								},
								MergePriority: &twenty,
							},
							{
								Name:          "x",
								Command:       "/path/to/x",
								MergePriority: &ten,
							},
						},
					},
				},
			},
		},
	}

	configFile := path.Join(testDir, "test-sugarkube-conf.yaml")
	ViperConfig.SetConfigFile(configFile)

	err := Load(ViperConfig)
	assert.Nil(t, err)
	assert.Equal(t, expectedConfig, CurrentConfig)
}
