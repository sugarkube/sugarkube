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
	"github.com/sugarkube/sugarkube/internal/pkg/interfaces"
	"github.com/sugarkube/sugarkube/internal/pkg/log"
	"github.com/sugarkube/sugarkube/internal/pkg/mock"
	"path/filepath"
	"testing"
)

func init() {
	log.ConfigureLogger("debug", false)
}

func getMockStackConfig(t *testing.T, dir string, name string, provider string, provisioner string,
	profile string, cluster string, region string, providerVarsDirs []string) interfaces.IStackConfig {

	return mock.Config{
		Name:             name,
		Provider:         provider,
		Provisioner:      provisioner,
		Profile:          profile,
		Cluster:          cluster,
		Region:           region,
		ProviderVarsDirs: providerVarsDirs,
		Dir:              dir,
	}
}

func TestLocalVarsDirs(t *testing.T) {
	stackObj := getMockStackConfig(t, "../../testdata/", "large", "local",
		"minikube", "local", "large", "fake-region", []string{"./stacks/"})

	assert.Equal(t, "local", stackObj.GetProvider())
	assert.Equal(t, []string{"./stacks/"}, stackObj.GetProviderVarsDirs())

	absTestDir, err := filepath.Abs("../../testdata")
	assert.Nil(t, err)

	expected := []string{
		filepath.Join(absTestDir, "stacks/local/profiles/local/clusters/large/values.yaml"),
	}

	providerObj := &LocalProvider{}
	actual, err := findVarsFiles(providerObj, stackObj)
	assert.Nil(t, err)

	assert.Equal(t, expected, actual, "Incorrect vars dirs returned")
}
