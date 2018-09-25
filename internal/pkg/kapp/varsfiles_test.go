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

func TestFindKappVarsFiles(t *testing.T) {

	absTestDir, err := filepath.Abs(testDir)
	assert.Nil(t, err)

	stackConfig := StackConfig{
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

	FindKappVarsFiles(absTestDir, &stackConfig, &stackConfig.Manifests[0].Kapps[0])
	assert.Equal(t, true, false)
	//assert.Equal(t, test.expected, test.input.Id)
}
