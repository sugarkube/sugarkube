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
	"path/filepath"
	"testing"
)

const testDir = "../../testdata"

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

func TestLoadStackConfig(t *testing.T) {
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
		Manifests: []Manifest{
			{
				Id:  "manifest1",
				Uri: "../../testdata/manifests/manifest1.yaml",
				Kapps: []Kapp{
					{
						Id:              "kappA",
						ShouldBePresent: true,
						Sources: []acquirer.Acquirer{
							acquirer.NewGitAcquirer(
								"pathA",
								"git@github.com:sugarkube/kapps-A.git",
								"kappA-0.1.0",
								"some/pathA",
								""),
						},
					},
				},
			},
			{
				Id:  "exampleManifest2",
				Uri: "../../testdata/manifests/manifest2.yaml",
				Kapps: []Kapp{
					{
						Id:              "kappC",
						ShouldBePresent: true,
						Sources: []acquirer.Acquirer{
							acquirer.NewGitAcquirer(
								"pathC",
								"git@github.com:sugarkube/kapps-C.git",
								"kappC-0.3.0",
								"some/pathC",
								""),
						},
					},
					{
						Id:              "kappB",
						ShouldBePresent: true,
						Sources: []acquirer.Acquirer{
							acquirer.NewGitAcquirer(
								"pathB",
								"git@github.com:sugarkube/kapps-B.git",
								"kappB-0.2.0",
								"some/pathB",
								""),
						},
					},
				},
				Options: ManifestOptions{
					Parallelisation: uint16(1),
				},
			},
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

func TestFindKappVarsFiles(t *testing.T) {

	absTestDir, err := filepath.Abs(testDir)
	assert.Nil(t, err)

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
			"./kapp-vars/", // todo - add some files here...
		},
		Manifests: []Manifest{
			{
				Id:  "manifest1",
				Uri: "../../testdata/manifests/manifest1.yaml",
				Kapps: []Kapp{
					{
						Id:              "kappA",
						ShouldBePresent: true,
						Sources: []acquirer.Acquirer{
							acquirer.NewGitAcquirer(
								"pathA",
								"git@github.com:sugarkube/kapps-A.git",
								"kappA-0.1.0",
								"some/pathA",
								""),
						},
					},
				},
			},
			{
				Id:  "exampleManifest2",
				Uri: "../../testdata/manifests/manifest2.yaml",
				Kapps: []Kapp{
					{
						Id:              "kappC",
						ShouldBePresent: true,
						Sources: []acquirer.Acquirer{
							acquirer.NewGitAcquirer(
								"pathC",
								"git@github.com:sugarkube/kapps-C.git",
								"kappC-0.3.0",
								"some/pathC",
								""),
						},
					},
					{
						Id:              "kappB",
						ShouldBePresent: true,
						Sources: []acquirer.Acquirer{
							acquirer.NewGitAcquirer(
								"pathB",
								"git@github.com:sugarkube/kapps-B.git",
								"kappB-0.2.0",
								"some/pathB",
								""),
						},
					},
				},
				Options: ManifestOptions{
					Parallelisation: uint16(1),
				},
			},
		},
	}

	expected := []string{
		filepath.Join(absTestDir, "kapp-vars/test-provider/test-provisioner/test-profile.yaml"),
		filepath.Join(absTestDir, "kapp-vars/test-provider/test-provisioner/test-account/values.yaml"),
		filepath.Join(absTestDir, "kapp-vars/test-provider/test-provisioner/test-account/test-region1/values.yaml"),
		filepath.Join(absTestDir, "kapp-vars/test-provider/test-provisioner/test-account/test-region1/kappA.yaml"),
	}

	results, err := stackConfig.FindKappVarsFiles(&stackConfig.Manifests[0].Kapps[0])
	assert.Nil(t, err)

	assert.Equal(t, expected, results)
}
