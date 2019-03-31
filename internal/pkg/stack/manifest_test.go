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
	"github.com/sugarkube/sugarkube/internal/pkg/config"
	"github.com/sugarkube/sugarkube/internal/pkg/installable"
	"github.com/sugarkube/sugarkube/internal/pkg/interfaces"
	"github.com/sugarkube/sugarkube/internal/pkg/log"
	"github.com/sugarkube/sugarkube/internal/pkg/structs"
	"os"
	"testing"
)

func init() {
	log.ConfigureLogger("debug", false)
}

// Helper to get acquirers in a single-valued context
func discardErrInstallable(installable interfaces.IInstallable, err error) interfaces.IInstallable {
	if err != nil {
		panic(err)
	}

	return installable
}

func TestValidateManifest(t *testing.T) {
	testManifestId := "testManifest"
	tests := []struct {
		name          string
		desc          string
		input         Manifest
		expectedError bool
	}{
		{
			name: "good",
			desc: "kapp IDs should be unique",
			input: Manifest{
				installables: []interfaces.IInstallable{
					discardErrInstallable(installable.New(testManifestId, []structs.KappDescriptorWithMaps{{Id: "example1"}})),
					discardErrInstallable(installable.New(testManifestId, []structs.KappDescriptorWithMaps{{Id: "example2"}})),
				},
			},
		},
		{
			name: "error_multiple_kapps_same_id",
			desc: "error when kapp IDs aren't unique",
			input: Manifest{
				installables: []interfaces.IInstallable{
					discardErrInstallable(installable.New(testManifestId, []structs.KappDescriptorWithMaps{{Id: "example1"}})),
					discardErrInstallable(installable.New(testManifestId, []structs.KappDescriptorWithMaps{{Id: "example2"}})),
					discardErrInstallable(installable.New(testManifestId, []structs.KappDescriptorWithMaps{{Id: "example1"}})),
				},
			},
		},
	}

	for _, test := range tests {
		err := ValidateManifest(&test.input)
		if test.expectedError {
			assert.NotNil(t, err)
		} else {
			assert.Nil(t, err)
		}
	}
}

func TestSetManifestDefaults(t *testing.T) {
	tests := []struct {
		name     string
		desc     string
		input    interfaces.IManifest
		expected string
	}{
		{
			name: "good",
			desc: "default manifest IDs should be the URI basename minus extension",
			input: &Manifest{
				descriptor: structs.ManifestDescriptor{Id: "", Uri: "example/manifest.yaml"},
			},
			expected: "manifest",
		},
	}

	for _, test := range tests {
		assert.Equal(t, test.expected, test.input.Id())
	}
}

func TestSelectKapps(t *testing.T) {

	// testing the correctness of this stack is handled in stack_test.go
	stackConfig, err := BuildStack("kops", "../../testdata/stacks.yaml",
		&structs.StackFile{}, "", &config.Config{}, os.Stdout)
	assert.Nil(t, err)
	assert.NotNil(t, stackConfig)

	includeSelector := []string{
		"exampleManifest2:kappA",
		"manifest1:kappA",
		"exampleManifest2:kappB",
	}

	var excludeSelector []string

	expectedKappIds := []string{
		"manifest1:kappA",
		"exampleManifest2:kappB",
		"exampleManifest2:kappA",
	}

	selectedKapps, err := SelectInstallables(stackConfig.GetConfig().Manifests(), includeSelector, excludeSelector)
	assert.Nil(t, err)

	assert.Equal(t, len(expectedKappIds), len(selectedKapps))

	for i := 0; i < len(expectedKappIds); i++ {
		assert.Equal(t, expectedKappIds[i], selectedKapps[i].FullyQualifiedId())
	}
}

func TestSelectKappsExclusions(t *testing.T) {

	// testing the correctness of this stack is handled in stack_test.go
	stack, err := BuildStack("kops", "../../testdata/stacks.yaml",
		&structs.StackFile{}, "", &config.Config{}, os.Stdout)
	assert.Nil(t, err)
	assert.NotNil(t, stack)

	includeSelector := []string{
		"exampleManifest2:*",
		"manifest1:kappA",
	}

	excludeSelector := []string{
		"exampleManifest2:kappA",
	}

	expectedKappIds := []string{
		"manifest1:kappA",
		"exampleManifest2:kappC",
		"exampleManifest2:kappB",
		"exampleManifest2:kappD",
	}

	selectedKapps, err := SelectInstallables(stack.GetConfig().Manifests(), includeSelector, excludeSelector)
	assert.Nil(t, err)

	for i := 0; i < len(expectedKappIds); i++ {
		assert.Equal(t, expectedKappIds[i], selectedKapps[i].FullyQualifiedId())
	}
}

//func TestParseManifestYaml(t *testing.T) {
//	manifest := structs.ManifestDescriptor{
//		Uri: "fake/uri",
//		Id:  "test-manifest",
//	}
//
//	tests := []struct {
//		name                 string
//		desc                 string
//		input                string
//		inputShouldBePresent bool
//		expectUnparsed       []KappDescriptor
//		expectAcquirers      [][]acquirer.Acquirer
//		expectedError        bool
//	}{
//		{
//			name: "good_parse",
//			desc: "check parsing acceptable input works",
//			input: `
//kapps:
//- id: example1
//  state: present
//  templates:
//    - source: example/template1.tpl
//      dest: example/dest.txt
//  sources:
//    - id: pathASpecial
//      uri: git@github.com:exampleA/repoA.git//example/pathA#branchA
//    - id: sampleNameB
//      uri: git@github.com:exampleB/repoB.git//example/pathB#branchB
//
//- id: example2
//  state: present
//  sources:
//  - uri: git@github.com:exampleA/repoA.git//example/pathA#branchA
//    options:
//      branch: new-branch
//  vars:
//    someVarA: valueA
//    someList:
//    - val1
//    - val2
//
//- id: example3
//  state: absent
//  sources:
//  - uri: git@github.com:exampleA/repoA.git//example/pathA#branchA
//  post_actions:
//  - cluster_update
//`,
//			expectUnparsed: []structs.KappDescriptorWithLists{
//				{
//					Id: "example1",
//					KappConfig: structs.KappConfig{
//						State: "present",
//						Templates: []structs.Template{
//							{
//								"example/template1.tpl",
//								"example/dest.txt",
//								false,
//							},
//						},
//					},
//					Sources: []structs.Source{
//						{Id: "pathASpecial",
//							Uri: "git@github.com:exampleA/repoA.git//example/pathA#branchA"},
//						{Id: "sampleNameB",
//							Uri: "git@github.com:exampleB/repoB.git//example/pathB#branchB"},
//					},
//				},
//				{
//					Id: "example2",
//					KappConfig: structs.KappConfig{
//						State: "present",
//						Vars: map[string]interface{}{
//							"someVarA": "valueA",
//							"someList": []interface{}{
//								"val1",
//								"val2",
//							},
//						},
//					},
//					Sources: []structs.Source{
//						{
//							Uri: "git@github.com:exampleA/repoA.git//example/pathA#branchA",
//							Options: map[string]interface{}{
//								"branch": "new-branch",
//							},
//						},
//					},
//				},
//				{
//					Id: "example3",
//					KappConfig: structs.KappConfig{
//						State: "absent",
//						PostActions: []string{
//							constants.TaskActionClusterUpdate,
//						},
//					},
//					Sources: []structs.Source{
//						{
//							Uri: "git@github.com:exampleA/repoA.git//example/pathA#branchA",
//						},
//					},
//				},
//			},
//			expectAcquirers: [][]acquirer.Acquirer{
//				{
//					// kapp1
//					discardErr(acquirer.NewGitAcquirer(
//						structs.Source{
//							Id:  "pathASpecial",
//							Uri: "git@github.com:exampleA/repoA.git//example/pathA#branchA",
//						},
//					)),
//					discardErr(acquirer.NewGitAcquirer(
//						structs.Source{
//							Id:  "sampleNameB",
//							Uri: "git@github.com:exampleB/repoB.git//example/pathB#branchB",
//						})),
//				},
//				// kapp 2
//				{
//					discardErr(acquirer.NewGitAcquirer(
//						structs.Source{
//							Uri: "git@github.com:exampleA/repoA.git//example/pathA#branchA",
//							Options: map[string]interface{}{
//								"branch": "new-branch",
//							},
//						})),
//				},
//				// kapp3
//				{
//					discardErr(acquirer.NewGitAcquirer(
//						structs.Source{
//							Uri: "git@github.com:exampleA/repoA.git//example/pathA#branchA",
//						},
//					)),
//				},
//			},
//			expectedError: false,
//		},
//	}
//
//	for _, test := range tests {
//		err := yaml.Unmarshal([]byte(test.input), &manifest)
//		assert.Nil(t, err)
//
//		if test.expectedError {
//			assert.NotNil(t, err)
//			assert.Nil(t, manifest.UnparsedKapps)
//		} else {
//			assert.Equal(t, test.expectUnparsed, manifest.UnparsedKapps, "unexpected conversion result for %s", test.name)
//			assert.Nil(t, err)
//
//			for i, parsedKapp := range manifest.ParsedKapps() {
//				log.Logger.Infof("%#v", parsedKapp)
//				acquirers, err := parsedKapp.Acquirers()
//				assert.Nil(t, err)
//				assert.Equal(t, test.expectAcquirers[i], acquirers, "unexpected acquirers for %s", test.name)
//			}
//		}
//	}
//
//	assert.NotEqual(t, manifest, Manifest{})
//}

// Test that overrides defined in a manifest file take effect
func TestManifestOverrides(t *testing.T) {

	// testing the correctness of this stack is handled elsewhere
	stackConfig, err := BuildStack("large", "../../testdata/stacks.yaml",
		&structs.StackFile{}, "", &config.Config{}, os.Stdout)
	assert.Nil(t, err)
	assert.NotNil(t, stackConfig)

	expectedDescriptor := structs.KappDescriptorWithMaps{
		Id: "kappA",
		KappConfig: structs.KappConfig{
			State: "absent",
			Vars: map[string]interface{}{
				"stackVar": "setInOverrides",
				"sizeVar":  "mediumOverridden",
				"colours": []interface{}{
					"red",
					"black",
				},
			},
		},
		Sources: map[string]structs.Source{
			"pathA": {
				Uri: "git@github.com:sugarkube/kapps-A.git//some/pathA#kappA-0.1.0",
				Options: map[string]interface{}{
					"branch": "stable",
				},
				IncludeValues: false,
			},
		},
		Output: map[string]structs.Output{},
	}

	actualDescriptor := stackConfig.GetConfig().Manifests()[0].Installables()[0].GetDescriptor()
	assert.Nil(t, err)

	assert.Equal(t, expectedDescriptor, actualDescriptor)
}

// Test that kapps with no overrides are correctly instantiated
func TestManifestOverridesNil(t *testing.T) {

	// testing the correctness of this stack is handled elsewhere
	stackConfig, err := BuildStack("large", "../../testdata/stacks.yaml",
		&structs.StackFile{}, "", &config.Config{}, os.Stdout)
	assert.Nil(t, err)
	assert.NotNil(t, stackConfig)

	expectedDescriptor := structs.KappDescriptorWithMaps{
		Id: "kappC",
		KappConfig: structs.KappConfig{
			State: "present",
			Vars:  map[string]interface{}{},
		},
		Sources: map[string]structs.Source{
			"special": {
				Id:            "special",
				Uri:           "git@github.com:sugarkube/kapps-C.git//kappC/some/special-path#kappC-0.3.0",
				Options:       map[string]interface{}{},
				IncludeValues: false,
			},
			"pathZ": {
				Uri:           "git@github.com:sugarkube/kapps-C.git//kappC/some/pathZ#kappZ-0.3.0",
				Options:       map[string]interface{}{},
				IncludeValues: false,
			},
			"pathX": {
				Uri:           "git@github.com:sugarkube/kapps-C.git//kappC/some/pathX#kappX-0.3.0",
				Options:       map[string]interface{}{},
				IncludeValues: false,
			},
			"pathY": {
				Uri:           "git@github.com:sugarkube/kapps-C.git//kappC/some/pathY#kappY-0.3.0",
				Options:       map[string]interface{}{},
				IncludeValues: false,
			},
		},
		Output: map[string]structs.Output{},
	}

	actualDescriptor := stackConfig.GetConfig().Manifests()[1].Installables()[0].GetDescriptor()
	assert.Nil(t, err)
	assert.Equal(t, expectedDescriptor, actualDescriptor)
}

//func TestApplyingManifestOverrides(t *testing.T) {
//
//	// testing the correctness of this stack is handled elsewhere
//	stackConfig, err := BuildStack("large", "../../testdata/stacks.yaml",
//		&structs.StackFile{}, "", &config.Config{}, os.Stdout)
//	assert.Nil(t, err)
//	assert.NotNil(t, stackConfig)
//
//	// in the actual manifest, the state is set to present but it's overridden
//	kappObj := stackConfig.GetConfig().Manifests()[0].Installables()[0]
//	assert.Equal(t, constants.AbsentKey, kappObj.State())
//	assert.Equal(t, map[string]interface{}{
//		"sizeVar":  "mediumOverridden",
//		"stackVar": "setInOverrides",
//		"colours": []interface{}{
//			"red",
//			"black",
//		}}, kappObj.Vars)
//
//	acquirers, err := kappObj.Acquirers()
//	assert.Nil(t, err)
//	assert.Equal(t, "git@github.com:sugarkube/kapps-A.git//some/pathA#stable", acquirers[0].Uri())
//}
