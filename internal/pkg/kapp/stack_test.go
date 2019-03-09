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

package kapp

import (
	"github.com/stretchr/testify/assert"
	"github.com/sugarkube/sugarkube/internal/pkg/acquirer"
	"gopkg.in/yaml.v2"
	"path/filepath"
	"testing"
)

const testDir = "../../testdata"

// Helper to get acquirers in a single-valued context
func discardErr(acquirer acquirer.Acquirer, err error) acquirer.Acquirer {
	if err != nil {
		panic(err)
	}

	return acquirer
}

func TestLoadStackConfigGarbagePath(t *testing.T) {
	_, err := LoadStackConfig("fake-path", "/fake/~/some?/~/garbage")
	assert.Error(t, err)
}

func TestLoadStackConfigNonExistentPath(t *testing.T) {
	_, err := LoadStackConfig("missing-path", "/missing/stacks.yaml")
	assert.Error(t, err)
}

func TestLoadStackConfigDir(t *testing.T) {
	_, err := LoadStackConfig("dir-path", "../../testdata")
	assert.Error(t, err)
}

func GetTestManifests() (*Manifest, *Manifest) {
	manifest1 := Manifest{
		ConfiguredId: "",
		Uri:          "../../testdata/manifests/manifest1.yaml",
		Overrides: map[string]interface{}{
			"kappA": map[interface{}]interface{}{
				"state": "absent",
				"sources": map[interface{}]interface{}{
					"pathA": map[interface{}]interface{}{
						"options": map[interface{}]interface{}{
							"branch": "stable",
						},
					},
				},
				"vars": map[interface{}]interface{}{
					"sizeVar":  "mediumOverridden",
					"stackVar": "setInOverrides",
				},
			},
		},
	}

	manifest1UnparsedKapps := []Kapp{
		{
			Id:       "kappA",
			State:    "present",
			manifest: nil,
			Sources: []acquirer.Source{
				{
					Uri: "git@github.com:sugarkube/kapps-A.git//some/pathA#kappA-0.1.0",
				},
			},
			Vars: map[string]interface{}{
				"sizeVar": "big",
				"colours": []interface{}{
					"red",
					"black",
				},
			},
		},
	}

	manifest1.UnparsedKapps = manifest1UnparsedKapps

	manifest2 := Manifest{
		ConfiguredId: "exampleManifest2",
		Uri:          "../../testdata/manifests/manifest2.yaml",
		Options: ManifestOptions{
			Parallelisation: uint16(1),
		},
	}

	manifest2UnparsedKapps := []Kapp{
		{
			Id:       "kappC",
			State:    "present",
			manifest: nil,
			Sources: []acquirer.Source{
				{
					Id:  "special",
					Uri: "git@github.com:sugarkube/kapps-C.git//kappC/some/special-path#kappC-0.3.0",
				},
				{Uri: "git@github.com:sugarkube/kapps-C.git//kappC/some/pathZ#kappZ-0.3.0"},
				{Uri: "git@github.com:sugarkube/kapps-C.git//kappC/some/pathX#kappX-0.3.0"},
				{Uri: "git@github.com:sugarkube/kapps-C.git//kappC/some/pathY#kappY-0.3.0"},
			},
		},
		{
			Id:       "kappB",
			State:    "present",
			manifest: nil,
			Sources: []acquirer.Source{
				{Uri: "git@github.com:sugarkube/kapps-B.git//some/pathB#kappB-0.2.0"},
			},
		},
		{
			Id:       "kappD",
			State:    "present",
			manifest: nil,
			Sources: []acquirer.Source{
				{
					Uri: "git@github.com:sugarkube/kapps-D.git//some/pathD#kappD-0.2.0",
					Options: map[string]interface{}{
						"branch": "kappDBranch",
					},
				},
			},
		},
		{
			Id:       "kappA",
			State:    "present",
			manifest: nil,
			Sources: []acquirer.Source{
				{IncludeValues: false,
					Uri: "git@github.com:sugarkube/kapps-A.git//some/pathA#kappA-0.2.0"},
			},
		},
	}

	manifest2.UnparsedKapps = manifest2UnparsedKapps

	return &manifest1, &manifest2
}

func TestLoadStackConfig(t *testing.T) {

	manifest1, manifest2 := GetTestManifests()

	expected := &StackConfig{
		Name:        "large",
		FilePath:    "../../testdata/stacks.yaml",
		Provider:    "local",
		Provisioner: "minikube",
		Profile:     "local",
		Cluster:     "large",
		ProviderVarsDirs: []string{
			"./stacks/",
		},
		TemplateDirs: []string{
			"templates1/",
			"templates2/",
		},
		Manifests: []*Manifest{
			manifest1,
			manifest2,
		},
		KappVarsDirs: []string{
			"sample-kapp-vars/",
		},
	}

	actual, err := LoadStackConfig("large", "../../testdata/stacks.yaml")
	assert.Nil(t, err)
	assert.Equal(t, expected, actual, "unexpected stack")
}

func TestLoadStackConfigMissingStackName(t *testing.T) {
	_, err := LoadStackConfig("missing-stack-name", "../../testdata/stacks.yaml")
	assert.Error(t, err)
}

func TestDir(t *testing.T) {
	stack := StackConfig{
		FilePath: "../../testdata/stacks.yaml",
	}

	expected := "../../testdata"
	actual := stack.Dir()

	assert.Equal(t, expected, actual, "Unexpected config dir")
}

// this should return the path to the current working dir, but it's difficult
// to meaningfully test.
func TestDirBlank(t *testing.T) {
	stack := StackConfig{}
	actual := stack.Dir()

	assert.NotNil(t, actual, "Unexpected config dir")
	assert.NotEmpty(t, actual, "Unexpected config dir")
}

func TestFindProviderVarsFiles(t *testing.T) {

	absTestDir, err := filepath.Abs(testDir)
	assert.Nil(t, err)

	manifest1, manifest2 := GetTestManifests()

	stackConfig := StackConfig{
		Name:        "large",
		FilePath:    "../../testdata/stacks.yaml",
		Provider:    "test-provider",
		Provisioner: "test-provisioner",
		Profile:     "test-profile",
		Cluster:     "test-cluster",
		Account:     "test-account",
		Region:      "region1",
		ProviderVarsDirs: []string{
			"./providers/",
		},
		Manifests: []*Manifest{
			manifest1,
			manifest2,
		},
	}

	expected := []string{
		filepath.Join(absTestDir, "providers/values.yaml"),
		filepath.Join(absTestDir, "providers/region1.yaml"),
		filepath.Join(absTestDir, "providers/test-provider/accounts/test-account/values.yaml"),
		filepath.Join(absTestDir, "providers/test-provider/accounts/test-account/region1.yaml"),
		filepath.Join(absTestDir, "providers/test-provider/accounts/test-account/profiles/test-profile/clusters/test-cluster/values.yaml"),
		filepath.Join(absTestDir, "providers/test-provider/accounts/test-account/profiles/test-profile/clusters/test-cluster/region1/values.yaml"),
		filepath.Join(absTestDir, "providers/test-account/region1.yaml"),
		filepath.Join(absTestDir, "providers/test-account/test-cluster/values.yaml"),
		filepath.Join(absTestDir, "providers/region1/values.yaml"),
		filepath.Join(absTestDir, "providers/region1/test-cluster.yaml"),
	}

	results, err := stackConfig.findProviderVarsFiles()
	assert.Nil(t, err)

	assert.Equal(t, expected, results)
}

func TestFindKappVarsFiles(t *testing.T) {

	absTestDir, err := filepath.Abs(testDir)
	assert.Nil(t, err)

	manifest1, manifest2 := GetTestManifests()

	stackConfig := StackConfig{
		Name:        "large",
		FilePath:    "../../testdata/stacks.yaml",
		Provider:    "test-provider",
		Provisioner: "test-provisioner",
		Profile:     "test-profile",
		Cluster:     "test-cluster",
		Account:     "test-account",
		Region:      "test-region1",
		ProviderVarsDirs: []string{
			"./stacks/",
		},
		KappVarsDirs: []string{
			"./sample-kapp-vars/kapp-vars/",
			"./sample-kapp-vars/kapp-vars2/",
		},
		Manifests: []*Manifest{
			manifest1,
			manifest2,
		},
	}

	expected := []string{
		filepath.Join(absTestDir, "sample-kapp-vars/kapp-vars/test-provider/test-provisioner/test-profile.yaml"),
		filepath.Join(absTestDir, "sample-kapp-vars/kapp-vars/test-provider/test-provisioner/test-account/values.yaml"),
		filepath.Join(absTestDir, "sample-kapp-vars/kapp-vars/test-provider/test-provisioner/test-account/test-region1/kappA.yaml"),
		filepath.Join(absTestDir, "sample-kapp-vars/kapp-vars/test-provider/test-provisioner/test-account/test-region1/values.yaml"),
		filepath.Join(absTestDir, "sample-kapp-vars/kapp-vars2/kappA.yaml"),
	}

	results, err := stackConfig.findVarsFiles(&stackConfig.Manifests[0].ParsedKapps()[0])
	assert.Nil(t, err)

	assert.Equal(t, expected, results)
}

func TestGetKappVarsFromFiles(t *testing.T) {

	manifest1, manifest2 := GetTestManifests()

	stackConfig := StackConfig{
		Name:        "large",
		FilePath:    "../../testdata/stacks.yaml",
		Provider:    "test-provider",
		Provisioner: "test-provisioner",
		Profile:     "test-profile",
		Cluster:     "test-cluster",
		Account:     "test-account",
		Region:      "test-region1",
		ProviderVarsDirs: []string{
			"./stacks/",
		},
		KappVarsDirs: []string{
			"./sample-kapp-vars",
			"./sample-kapp-vars/kapp-vars/",
			"./sample-kapp-vars/kapp-vars2/",
		},
		Manifests: []*Manifest{
			manifest1,
			manifest2,
		},
	}

	expected := `globals:
  account: test-account-val
kapp: kappA-val
kappASisterDir: extra-val
kappOverride: kappA-val-override
profile: test-profile-val
region: test-region1-val
regionOverride: region-val-override
`

	results, err := stackConfig.getVarsFromFiles(&stackConfig.Manifests[0].ParsedKapps()[0])
	assert.Nil(t, err)

	yamlResults, err := yaml.Marshal(results)
	assert.Nil(t, err)

	assert.Equal(t, expected, string(yamlResults[:]))
}

func TestAllManifests(t *testing.T) {
	stackConfig := StackConfig{
		Manifests: []*Manifest{
			{
				UnparsedKapps: []Kapp{
					{
						Id: "kapp2",
					},
				},
			},
			{
				UnparsedKapps: []Kapp{
					{
						Id: "kapp3",
					},
				},
			},
		},
	}

	assert.Equal(t, 2, len(stackConfig.Manifests))
}
