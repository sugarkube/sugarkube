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
	"github.com/sugarkube/sugarkube/internal/pkg/structs"
	"gopkg.in/yaml.v2"
	"testing"
)

func TestLoadStackConfigGarbagePath(t *testing.T) {
	_, err := loadStackFile("fake-path", "/fake/~/some?/~/garbage")
	assert.Error(t, err)
}

func TestLoadStackConfigNonExistentPath(t *testing.T) {
	_, err := loadStackFile("missing-path", "/missing/stacks.yaml")
	assert.Error(t, err)
}

func TestLoadStackConfigDir(t *testing.T) {
	_, err := loadStackFile("dir-path", "../../testdata")
	assert.Error(t, err)
}

func GetTestManifestDescriptors() []structs.ManifestDescriptor {
	manifest1 := structs.ManifestDescriptor{
		Id:  "",
		Uri: "../../testdata/manifests/manifest1.yaml",
		Overrides: map[string]structs.KappDescriptorWithMaps{
			"kappA": {
				KappConfig: structs.KappConfig{
					State: "absent",
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

	//manifest1KappDescriptors := []structs.KappDescriptorWithLists{
	//	{
	//		Id: "kappA",
	//		KappConfig: structs.KappConfig{
	//			State: "present",
	//			Vars: map[string]interface{}{
	//				"sizeVar": "big",
	//				"colours": []interface{}{
	//					"red",
	//					"black",
	//				},
	//			},
	//		},
	//		Sources: []structs.Source{
	//			{
	//				Uri: "git@github.com:sugarkube/kapps-A.git//some/pathA#kappA-0.1.0",
	//			},
	//		},
	//	},
	//}
	//
	//manifest1.UnparsedKapps = manifest1UnparsedKapps

	manifest2 := structs.ManifestDescriptor{
		Id:  "exampleManifest2",
		Uri: "../../testdata/manifests/manifest2.yaml",
		//Options: ManifestOptions{
		//	Parallelisation: uint16(1),
		//},
	}

	//manifest2KappDescriptors := []structs.KappDescriptorWithLists{
	//	{
	//		Id: "kappC",
	//		KappConfig: structs.KappConfig{
	//			State: "present",
	//		},
	//		Sources: []structs.Source{
	//			{
	//				Id:  "special",
	//				Uri: "git@github.com:sugarkube/kapps-C.git//kappC/some/special-path#kappC-0.3.0",
	//			},
	//			{Uri: "git@github.com:sugarkube/kapps-C.git//kappC/some/pathZ#kappZ-0.3.0"},
	//			{Uri: "git@github.com:sugarkube/kapps-C.git//kappC/some/pathX#kappX-0.3.0"},
	//			{Uri: "git@github.com:sugarkube/kapps-C.git//kappC/some/pathY#kappY-0.3.0"},
	//		},
	//	},
	//	{
	//		Id: "kappB",
	//		KappConfig: structs.KappConfig{
	//			State: "present",
	//		},
	//		Sources: []structs.Source{
	//			{Uri: "git@github.com:sugarkube/kapps-B.git//some/pathB#kappB-0.2.0"},
	//		},
	//	},
	//	{
	//		Id: "kappD",
	//		KappConfig: structs.KappConfig{
	//			State: "present",
	//		},
	//		Sources: []structs.Source{
	//			{
	//				Uri: "git@github.com:sugarkube/kapps-D.git//some/pathD#kappD-0.2.0",
	//				Options: map[string]interface{}{
	//					"branch": "kappDBranch",
	//				},
	//			},
	//		},
	//	},
	//	{
	//		Id: "kappA",
	//		KappConfig: structs.KappConfig{
	//			State: "present",
	//		},
	//		Sources: []structs.Source{
	//			{IncludeValues: false,
	//				Uri: "git@github.com:sugarkube/kapps-A.git//some/pathA#kappA-0.2.0"},
	//		},
	//	},
	//}
	//
	//manifest2.UnparsedKapps = manifest2UnparsedKapps

	return []structs.ManifestDescriptor{manifest1, manifest2}
}

func TestLoadStackConfig(t *testing.T) {

	expected := &structs.StackFile{
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
		ManifestDescriptors: GetTestManifestDescriptors(),
		KappVarsDirs: []string{
			"sample-kapp-vars/",
		},
	}

	actual, err := loadStackFile("large", "../../testdata/stacks.yaml")
	assert.Nil(t, err)
	assert.Equal(t, expected, actual, "unexpected stack")
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

	stackFile := structs.StackFile{
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
		ManifestDescriptors: GetTestManifestDescriptors(),
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
