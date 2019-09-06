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
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/sugarkube/sugarkube/internal/pkg/interfaces"
	"github.com/sugarkube/sugarkube/internal/pkg/structs"
	"gopkg.in/yaml.v2"
	"path/filepath"
	"testing"
)

const testDir = "../../testdata"

func TestLoadStackConfigGarbagePath(t *testing.T) {
	_, err := loadStackFile("fake-path", "/fake/~/some?/~/garbage")
	assert.Error(t, err)
}

func TestLoadStackConfigNonExistentPath(t *testing.T) {
	_, err := loadStackFile("missing-path", "/missing/stacks.yaml")
	assert.Error(t, err)
}

func TestLoadStackConfigDir(t *testing.T) {
	_, err := loadStackFile("dir-path", testDir)
	assert.Error(t, err)
}

func GetTestManifestDescriptors() []structs.ManifestDescriptor {
	descriptor1 := structs.ManifestDescriptor{
		Id:  "",
		Uri: "manifests/manifest1.yaml",
		Overrides: map[string]structs.KappDescriptorWithMaps{
			"kappA": {
				KappConfig: structs.KappConfig{
					Vars: map[string]interface{}{
						"sizeVar":  "mediumOverridden",
						"stackVar": "setInOverrides",
					},
				},
				Sources: map[string]structs.Source{
					"pathA": {
						Options: map[string]interface{}{
							"branch": "stable",
						},
					},
				},
			},
		},
	}

	descriptor2 := structs.ManifestDescriptor{
		Id:  "exampleManifest2",
		Uri: "manifests/manifest2.yaml",
	}

	return []structs.ManifestDescriptor{descriptor1, descriptor2}
}

func GetTestManifests(t *testing.T) []interfaces.IManifest {
	absTestDir, err := filepath.Abs(testDir)
	assert.Nil(t, err)

	descriptor1 := structs.ManifestDescriptor{
		Id:  "",
		Uri: filepath.Join(absTestDir, "manifests/manifest1.yaml"),
		Overrides: map[string]structs.KappDescriptorWithMaps{
			"kappA": {
				KappConfig: structs.KappConfig{
					Vars: map[string]interface{}{
						"sizeVar":  "mediumOverridden",
						"stackVar": "setInOverrides",
					},
				},
				Sources: map[string]structs.Source{
					"pathA": {
						Options: map[string]interface{}{
							"branch": "stable",
						},
					},
				},
			},
		},
	}

	manifest1KappDescriptors := []structs.KappDescriptorWithLists{
		{
			Id: "kappA",
			KappConfig: structs.KappConfig{
				Vars: map[string]interface{}{
					"sizeVar": "big",
					"colours": []interface{}{
						"red",
						"black",
					},
				},
			},
			Sources: []structs.Source{
				{
					Uri: "git@github.com:sugarkube/kapps-A.git//some/pathA#kappA-0.1.0",
				},
			},
		},
	}

	manifest1 := Manifest{
		descriptor: descriptor1,
		manifestFile: structs.ManifestFile{
			KappDescriptor: manifest1KappDescriptors,
			Defaults: structs.KappConfig{
				Vars: map[string]interface{}{
					"namespace": "test-namespace",
				},
			},
		},
	}

	descriptor2 := structs.ManifestDescriptor{
		Id:  "exampleManifest2",
		Uri: filepath.Join(absTestDir, "manifests/manifest2.yaml"),
	}

	manifest2KappDescriptors := []structs.KappDescriptorWithLists{
		{
			Id:         "kappC",
			KappConfig: structs.KappConfig{},
			Sources: []structs.Source{
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
			Id:         "kappB",
			KappConfig: structs.KappConfig{},
			Sources: []structs.Source{
				{Uri: "git@github.com:sugarkube/kapps-B.git//some/pathB#kappB-0.2.0"},
			},
		},
		{
			Id:         "kappD",
			KappConfig: structs.KappConfig{},
			Sources: []structs.Source{
				{
					Uri: "git@github.com:sugarkube/kapps-D.git//some/pathD#kappD-0.2.0",
					Options: map[string]interface{}{
						"branch": "kappDBranch",
					},
				},
			},
		},
		{
			Id:         "kappA",
			KappConfig: structs.KappConfig{},
			Sources: []structs.Source{
				{Uri: "git@github.com:sugarkube/kapps-A.git//some/pathA#kappA-0.2.0"},
			},
		},
	}

	manifest2 := Manifest{
		descriptor: descriptor2,
		manifestFile: structs.ManifestFile{
			KappDescriptor: manifest2KappDescriptors,
			Options: structs.ManifestOptions{
				IsSequential: true,
			},
		},
	}

	return []interfaces.IManifest{&manifest1, &manifest2}
}

func TestLoadStackConfig(t *testing.T) {

	absFilePath, err := filepath.Abs("../../testdata/stacks.yaml")
	assert.Nil(t, err)

	expected := &structs.StackFile{
		Name:        "large",
		FilePath:    absFilePath,
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
		ManifestDescriptors: GetTestManifestDescriptors(),
		KappVarsDirs: []string{
			"sample-kapp-vars/",
		},
	}

	stackFile, err := loadStackFile("large", "../../testdata/stacks.yaml")
	assert.Nil(t, err)
	assert.Equal(t, expected, stackFile, "unexpected stack")

	stackConfig, err := parseStackFile(*stackFile)
	assert.Nil(t, err)

	expectedManifests := GetTestManifests(t)
	actualManifests := stackConfig.Manifests()

	for i := 0; i < len(expectedManifests); i++ {
		// blank the installables - we'll test loading those elsewhere
		actualManifests[i].(*Manifest).installables = nil
		assert.Equal(t, expectedManifests[i], actualManifests[i],
			fmt.Sprintf("Manifest at index %d doesn't match", i))
	}
}

func TestLoadStackConfigMissingStackName(t *testing.T) {
	_, err := loadStackFile("missing-stack-name", "../../testdata/stacks.yaml")
	assert.Error(t, err)
}

func TestDir(t *testing.T) {
	stackConfig := &StackConfig{
		stackFile: structs.StackFile{
			FilePath: "../../testdata/stacks.yaml",
		},
	}

	expected := "../../testdata"
	actual := stackConfig.GetDir()

	assert.Equal(t, expected, actual, "Unexpected config dir")
}

// this should return the path to the current working dir, but it's difficult
// to meaningfully test.
func TestDirBlank(t *testing.T) {
	stack := StackConfig{}
	actual := stack.GetDir()

	assert.NotNil(t, actual, "Unexpected config dir")
	assert.NotEmpty(t, actual, "Unexpected config dir")
}

func TestGetKappVarsFromFiles(t *testing.T) {

	absFilePath, err := filepath.Abs("../../testdata/stacks.yaml")
	assert.Nil(t, err)

	stackFile := structs.StackFile{
		Name:        "large",
		FilePath:    absFilePath,
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
		ManifestDescriptors: GetTestManifestDescriptors(),
	}

	expected := `kapp:
  cacheRoot: ""
  id: kappA
  templates: {}
  vars:
    colours:
    - red
    - black
    globals:
      account: test-account-val
    kapp: kappA-val
    kappASisterDir: extra-val
    kappOverride: kappA-val-override
    namespace: test-namespace
    profile: test-profile-val
    region: test-region1-val
    regionOverride: region-val-override
    sizeVar: mediumOverridden
    stackVar: setInOverrides
`

	stackConfig, err := parseStackFile(stackFile)
	assert.Nil(t, err)

	stackObj := &Stack{
		config: stackConfig,
	}

	kappObj := stackConfig.Manifests()[0].Installables()[0]
	results, err := kappObj.Vars(stackObj)
	assert.Nil(t, err)

	yamlResults, err := yaml.Marshal(results)
	assert.Nil(t, err)

	assert.Equal(t, expected, string(yamlResults[:]))
}
