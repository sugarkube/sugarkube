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

package provider

import (
	"github.com/stretchr/testify/assert"
	"github.com/sugarkube/sugarkube/internal/pkg/config"
	"github.com/sugarkube/sugarkube/internal/pkg/interfaces"
	"github.com/sugarkube/sugarkube/internal/pkg/stack"
	"github.com/sugarkube/sugarkube/internal/pkg/structs"
	"os"
	"path/filepath"
	"testing"
)

const testDir = "../../testdata"

func loadStack(t *testing.T) interfaces.IStack {
	stackObj, err := stack.BuildStack("large", "../../testdata/stacks.yaml",
		&structs.Stack{}, &config.Config{}, os.Stdout)
	assert.Nil(t, err)

	return stackObj
}

func TestStackConfigVars(t *testing.T) {
	stackObj := loadStack(t)

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

	providerImpl, err := newProviderImpl(stackObj.GetConfig().Provider(), stackObj.GetConfig())
	assert.Nil(t, err)

	actual, err := GetVarsFromFiles(providerImpl, stackObj.GetConfig())
	assert.Nil(t, err)
	assert.Equal(t, expected, actual, "Mismatching vars")
}

func TestNewNonExistentProvider(t *testing.T) {
	stackObj := loadStack(t)

	actual, err := newProviderImpl("bananas", stackObj.GetConfig())
	assert.NotNil(t, err)
	assert.Nil(t, actual)
}

func TestNewProviderError(t *testing.T) {
	stackObj := loadStack(t)
	actual, err := newProviderImpl("nonsense", stackObj.GetConfig())
	assert.NotNil(t, err)
	assert.Nil(t, actual)
}

func TestNewLocalProvider(t *testing.T) {
	stackObj := loadStack(t)
	actual, err := newProviderImpl(LOCAL, stackObj.GetConfig())
	assert.Nil(t, err)
	assert.Equal(t, &LocalProvider{}, actual)
}

func TestNewAWSProvider(t *testing.T) {
	stackObj := loadStack(t)
	actual, err := newProviderImpl(AWS, stackObj.GetConfig())
	assert.Nil(t, err)
	assert.Equal(t, &AwsProvider{}, actual)
}

func TestFindProviderVarsFiles(t *testing.T) {

	absTestDir, err := filepath.Abs(testDir)
	assert.Nil(t, err)

	stackConfig := structs.Stack{
		Name:        "large",
		FilePath:    "../../testdata/stacks.yaml",
		Provider:    "aws",
		Provisioner: "test-provisioner",
		Profile:     "test-profile",
		Cluster:     "test-cluster",
		Account:     "test-account",
		Region:      "region1",
		ProviderVarsDirs: []string{
			"./providers/",
		},
	}

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

	providerImpl, err := New(&stackConfig)
	assert.Nil(t, err)

	results, err := findVarsFiles(providerImpl, &stackConfig)
	assert.Nil(t, err)

	assert.Equal(t, expected, results)
}
