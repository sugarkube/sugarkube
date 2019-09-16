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

package provider

import (
	"github.com/stretchr/testify/assert"
	"github.com/sugarkube/sugarkube/internal/pkg/mock"
	"path/filepath"
	"testing"
)

func TestStackConfigVars(t *testing.T) {
	stackObj := mock.GetMockStack(t, testDir, "large", "", "local",
		"minikube", "local", "large", "fake-region", []string{"./stacks/"})

	expected := map[string]interface{}{
		"provisioner": map[interface{}]interface{}{
			"binary": "minikube",
			"params": map[interface{}]interface{}{
				"start": map[interface{}]interface{}{
					"disk_size": "120g",
					"memory":    4096,
					"cpus":      4,
				},
			},
		},
	}

	providerImpl, err := New(stackObj.GetConfig())
	assert.Nil(t, err)

	actual, err := GetVarsFromFiles(providerImpl, stackObj.GetConfig())
	assert.Nil(t, err)
	assert.Equal(t, expected, actual, "Mismatching vars")
}

func TestNewNonExistentProvider(t *testing.T) {
	stackObj := mock.GetMockStackConfig(t, testDir, "large", "", "bananas",
		"minikube", "local", "large", "fake-region", []string{"./stacks/"})

	actual, err := New(stackObj)
	assert.NotNil(t, err)
	assert.Nil(t, actual)
}

func TestNewLocalProvider(t *testing.T) {
	stackObj := mock.GetMockStackConfig(t, testDir, "large", "", "local",
		"minikube", "local", "large", "fake-region", []string{"./stacks/"})

	actual, err := New(stackObj)
	assert.Nil(t, err)
	assert.Equal(t, &LocalProvider{varsPaths: []string{}}, actual)
}

func TestNewAWSProvider(t *testing.T) {
	stackObj := mock.GetMockStackConfig(t, testDir, "large", "", "aws",
		"minikube", "local", "large", "fake-region", []string{"./stacks/"})
	actual, err := New(stackObj)
	assert.Nil(t, err)
	assert.Equal(t, &AwsProvider{
		region:    "fake-region",
		varsPaths: []string{},
	}, actual)
}

func TestFindProviderVarsFiles(t *testing.T) {

	absTestDir, err := filepath.Abs(testDir)
	assert.Nil(t, err)

	stackObj := mock.GetMockStackConfig(t, testDir, "large", "test-account", "aws",
		"test-provisioner", "test-profile", "test-cluster", "region1",
		[]string{"./providers/"})

	expected := []string{
		filepath.Join(absTestDir, "providers/values.yaml"),
		filepath.Join(absTestDir, "providers/region1.yaml"),
		filepath.Join(absTestDir, "providers/aws/accounts/test-account/values.yaml"),
		filepath.Join(absTestDir, "providers/aws/accounts/test-account/region1.yaml"),
		filepath.Join(absTestDir, "providers/aws/accounts/test-account/profiles/test-profile/clusters/test-cluster/values.yaml"),
		filepath.Join(absTestDir, "providers/aws/accounts/test-account/profiles/test-profile/clusters/test-cluster/region1/values.yaml"),
		filepath.Join(absTestDir, "providers/test-account/region1.yaml"),
		filepath.Join(absTestDir, "providers/test-account/test-cluster/values.yaml"),
		filepath.Join(absTestDir, "providers/region1/values.yaml"),
		filepath.Join(absTestDir, "providers/region1/test-cluster.yaml"),
	}

	providerImpl, err := New(stackObj)
	assert.Nil(t, err)

	results, err := findVarsFiles(providerImpl, stackObj)
	assert.Nil(t, err)

	assert.Equal(t, expected, results)
}
