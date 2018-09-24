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

package installer

import (
	"github.com/stretchr/testify/assert"
	"github.com/sugarkube/sugarkube/internal/pkg/kapp"
	"path"
	"path/filepath"
	"strings"
	"testing"
)

// Test against testdata
func TestGetCliArgs(t *testing.T) {

	absTestDir, err := filepath.Abs(testDir)
	assert.Nil(t, err)

	parameteriser := Parameteriser{
		Name: IMPLEMENTS_HELM,
		kappObj: &kapp.Kapp{
			RootDir: path.Join(absTestDir, "sample-chart"),
		},
	}

	tests := []struct {
		name         string
		desc         string
		stackConfig  kapp.StackConfig
		expectValues string
	}{
		{
			name: "aws",
			desc: "test that files are found in the correct order",
			stackConfig: kapp.StackConfig{
				Provider: "aws",
				Account:  "dev",
				Profile:  "dev",
				Cluster:  "dev1",
				Region:   "eu-west-1",
			},
			expectValues: "helm-opts=-f {kappDir}/values-aws.yaml " +
				"-f {kappDir}/values-dev.yaml -f {kappDir}/values-dev1.yaml " +
				"-f {kappDir}/values-eu-west-1.yaml",
		},
		{
			name: "local",
			desc: "test that files are found in the correct order",
			stackConfig: kapp.StackConfig{
				Provider: "local",
				Profile:  "dev",
				Cluster:  "dev1",
			},
			expectValues: "helm-opts=-f {kappDir}/values-dev.yaml " +
				"-f {kappDir}/values-dev1.yaml",
		},
	}

	for _, test := range tests {
		configSubstrings := []string{
			test.stackConfig.Provider,
			test.stackConfig.Account, // may be blank depending on the provider
			test.stackConfig.Profile,
			test.stackConfig.Cluster,
			test.stackConfig.Region, // may be blank depending on the provider
		}

		result, err := parameteriser.GetCliArgs(configSubstrings)
		assert.Nil(t, err)
		expected := strings.Replace(test.expectValues, "{kappDir}", parameteriser.kappObj.RootDir, -1)
		assert.Equal(t, expected, result, "unexpected files returned for %s", test.name)
	}
}
