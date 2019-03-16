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

import (
	"github.com/stretchr/testify/assert"
	"github.com/sugarkube/sugarkube/internal/pkg/kapp"
	"github.com/sugarkube/sugarkube/internal/pkg/log"
	"github.com/sugarkube/sugarkube/internal/pkg/registry"
	"testing"
)

func init() {
	log.ConfigureLogger("debug", false)
}

// Test that registry values override values when returning templated vars
func TestTemplatedVarsWithRegistry(t *testing.T) {

	expectedVarsBlankRegistry := map[string]interface{}{
		"sugarkube": map[interface{}]interface{}{
			"defaultVars": []interface{}{"", "", "", "", "testRegion"}},
		"someKey1": "valueA",
		"someKey2": "valueB",
		"someKey3": "valueC",
		"stack": map[interface{}]interface{}{
			"filePath":    "",
			"name":        "",
			"profile":     "",
			"provider":    "",
			"provisioner": "",
			"region":      "testRegion",
			"account":     "",
			"cluster":     "",
		},
	}

	expectedVarsUpdatedRegistry := map[string]interface{}{
		"sugarkube": map[interface{}]interface{}{
			"defaultVars": []interface{}{"", "", "", "", "testRegion"}},
		"someKey1": "valueA",
		"someKey2": "updatedValue",
		"someKey3": "valueC",
		"stack": map[interface{}]interface{}{
			"filePath":    "",
			"name":        "",
			"profile":     "",
			"provider":    "",
			"provisioner": "",
			"region":      "testRegion",
			"account":     "",
			"cluster":     "",
		},
	}

	registryObj := registry.NewRegistry()

	stackConfig := &kapp.StackConfig{
		Region: "testRegion",
	}

	stackConfig.SetProviderVars(map[string]interface{}{
		"someKey1": "valueA",
		"someKey2": "valueB",
		"someKey3": "valueC",
	})

	stackObj := &Stack{
		Config:      stackConfig,
		Provider:    nil,
		Provisioner: nil,
		Status: &ClusterStatus{
			isOnline:              false,
			isReady:               false,
			sleepBeforeReadyCheck: 0,
			startedThisRun:        false,
		},
		registry: &registryObj,
	}

	templatedVars, err := stackObj.TemplatedVars(nil, map[string]interface{}{})
	assert.Nil(t, err)
	assert.Equal(t, expectedVarsBlankRegistry, templatedVars)

	// update the registry to override a value
	registryObj.SetString("someKey2", "updatedValue")
	templatedVars, err = stackObj.TemplatedVars(nil, map[string]interface{}{})
	assert.Nil(t, err)
	assert.Equal(t, expectedVarsUpdatedRegistry, templatedVars)
}
