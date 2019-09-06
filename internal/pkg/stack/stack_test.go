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

package stack

import (
	"github.com/stretchr/testify/assert"
	"github.com/sugarkube/sugarkube/internal/pkg/constants"
	"github.com/sugarkube/sugarkube/internal/pkg/log"
	"github.com/sugarkube/sugarkube/internal/pkg/registry"
	"github.com/sugarkube/sugarkube/internal/pkg/structs"
	"os"
	"testing"
)

func init() {
	log.ConfigureLogger("debug", false, os.Stderr)
}

// Test that registry values override values when returning templated vars
func TestTemplatedVarsWithRegistry(t *testing.T) {

	expectedVarsBlankRegistry := map[string]interface{}{
		"sugarkube": map[interface{}]interface{}{
			"defaultVars": []interface{}{"", "", "testAccount", "", "", "testRegion"}},
		"someKey1":                      "valueA",
		"someKey2":                      "valueB",
		"someKey3":                      "valueC",
		constants.RegistryKeyKubeConfig: "",
		"stack": map[interface{}]interface{}{
			"filePath":    "",
			"name":        "",
			"profile":     "",
			"provider":    "",
			"provisioner": "",
			"region":      "testRegion",
			"account":     "testAccount",
			"cluster":     "",
		},
	}

	expectedVarsUpdatedRegistry := map[string]interface{}{
		"sugarkube": map[interface{}]interface{}{
			"defaultVars": []interface{}{"", "", "testAccount", "", "", "testRegion"}},
		"someKey1":                      "valueA",
		"someKey2":                      "updatedValue",
		"someKey3":                      "valueC",
		constants.RegistryKeyKubeConfig: "",
		"stack": map[interface{}]interface{}{
			"filePath":    "",
			"name":        "",
			"profile":     "",
			"provider":    "",
			"provisioner": "",
			"region":      "testRegion",
			"account":     "testAccount",
			"cluster":     "",
		},
	}

	registryObj := registry.New()

	stackConfig := &StackConfig{
		stackFile: structs.StackFile{
			Region:  "testRegion",
			Account: "testAccount",
		},
	}

	stackConfig.SetProviderVars(map[string]interface{}{
		"someKey1": "valueA",
		"someKey2": "valueB",
		"someKey3": "valueC",
	})

	stackObj := &Stack{
		config:      stackConfig,
		provider:    nil,
		provisioner: nil,
		status: &ClusterStatus{
			isOnline:              false,
			isReady:               false,
			sleepBeforeReadyCheck: 0,
			startedThisRun:        false,
		},
		registry: registryObj,
	}

	templatedVars, err := stackObj.GetTemplatedVars(nil, map[string]interface{}{})
	assert.Nil(t, err)
	assert.Equal(t, expectedVarsBlankRegistry, templatedVars)

	// update the registry to override a value
	err = registryObj.Set("someKey2", "updatedValue")
	assert.Nil(t, err)
	templatedVars, err = stackObj.GetTemplatedVars(nil, map[string]interface{}{})
	assert.Nil(t, err)
	assert.Equal(t, expectedVarsUpdatedRegistry, templatedVars)
}
