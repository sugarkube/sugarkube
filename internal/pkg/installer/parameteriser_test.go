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
	"github.com/sugarkube/sugarkube/internal/pkg/acquirer"
	"github.com/sugarkube/sugarkube/internal/pkg/kapp"
	"github.com/sugarkube/sugarkube/internal/pkg/provider"
	"path"
	"path/filepath"
	"strings"
	"testing"
)

// Test against testdata
func TestGetCliArgs(t *testing.T) {

	absTestDir, err := filepath.Abs(testDir)
	assert.Nil(t, err)

	kappObj := &kapp.Kapp{
		Id: "sample-kapp",
		Sources: []acquirer.Acquirer{
			acquirer.NewGitAcquirer(
				"sample-chart",
				"git@github.com:helm/charts.git",
				"master",
				"/fake/chart",
				""),
			acquirer.NewGitAcquirer(
				"sister-dir",
				"git@github.com:helm/charts.git",
				"master",
				"/fake/sister",
				"false"),
		},
	}

	manifest := kapp.Manifest{
		Id: "sample-manifest",
	}

	kappObj.SetManifest(&manifest)
	kappObj.SetCacheDir(path.Join(absTestDir, "sample-cache"))

	tfStackConfig := kapp.StackConfig{
		Provider:         "local",
		Profile:          "local",
		Cluster:          "large",
		ProviderVarsDirs: []string{filepath.Join(absTestDir, "stacks")},
	}
	tfProviderImpl, err := provider.NewProvider(&tfStackConfig)
	assert.Nil(t, err)

	tests := []struct {
		name          string
		desc          string
		parameteriser Parameteriser
		stackConfig   kapp.StackConfig
		expectValues  string
	}{
		{
			name: "aws",
			desc: "test that files are found in the correct order",
			parameteriser: Parameteriser{
				Name:    IMPLEMENTS_HELM,
				kappObj: kappObj,
			},
			stackConfig: kapp.StackConfig{
				Provider: "zaws", // prepend a 'z' otherwise results
				Account:  "dev",  // will just be alphabetical
				Profile:  "dev",
				Cluster:  "dev1",
				Region:   "eu-west-1",
			},
			expectValues: "helm-opts=-f {kappDir}/values-zaws.yaml " +
				"-f {kappDir}/values-dev.yaml -f {kappDir}/values-dev1.yaml " +
				"-f {kappDir}/values-eu-west-1.yaml",
		},
		{
			name: "local",
			desc: "test that files are found in the correct order",
			parameteriser: Parameteriser{
				Name:    IMPLEMENTS_HELM,
				kappObj: kappObj,
			},
			stackConfig: kapp.StackConfig{
				Provider: "local",
				Profile:  "dev",
				Cluster:  "dev1",
			},
			expectValues: "helm-opts=-f {kappDir}/values-dev.yaml " +
				"-f {kappDir}/values-dev1.yaml",
		},
		{
			name: "terraform",
			desc: "test that terraform files are found in the correct order",
			parameteriser: Parameteriser{
				Name:         IMPLEMENTS_TERRAFORM,
				kappObj:      kappObj,
				providerImpl: &tfProviderImpl,
			},
			stackConfig: tfStackConfig,
			expectValues: "tf-opts=-var-file {kappDir}/terraform_local/vars/local.tfvars " +
				"-var-file {kappDir}/terraform_local/vars/large.tfvars",
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

		result, err := test.parameteriser.GetCliArgs(configSubstrings)
		assert.Nil(t, err)

		// the dir of the actual chart inside the cache dir
		chartDir := path.Join(test.parameteriser.kappObj.CacheDir(), "sample-chart")

		expected := strings.Replace(test.expectValues, "{kappDir}", chartDir, -1)
		assert.Equal(t, expected, result, "unexpected files returned for %s",
			test.name)
	}
}
