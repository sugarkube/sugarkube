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
	"github.com/sugarkube/sugarkube/internal/pkg/constants"
	"github.com/sugarkube/sugarkube/internal/pkg/log"
	"gopkg.in/yaml.v2"
	"testing"
)

func init() {
	log.ConfigureLogger("debug", false)
}

func TestParseManifestYaml(t *testing.T) {
	manifest := Manifest{
		Uri:          "fake/uri",
		ConfiguredId: "test-manifest",
	}

	tests := []struct {
		name                 string
		desc                 string
		input                string
		inputShouldBePresent bool
		expectUnparsed       []Kapp
		expectAcquirers      [][]acquirer.Acquirer
		expectedError        bool
	}{
		{
			name: "good_parse",
			desc: "check parsing acceptable input works",
			input: `
kapps:
  - id: example1
    state: present
    templates:        
      - source: example/template1.tpl
        dest: example/dest.txt
    sources:
      - id: pathASpecial
        uri: git@github.com:exampleA/repoA.git//example/pathA#branchA
      - id: sampleNameB
        uri: git@github.com:exampleB/repoB.git//example/pathB#branchB

  - id: example2
    state: present
    sources:
    - uri: git@github.com:exampleA/repoA.git//example/pathA#branchA
      options:
        branch: new-branch
    vars:
      someVarA: valueA
      someList:
      - val1
      - val2

  - id: example3
    state: absent
    sources:
    - uri: git@github.com:exampleA/repoA.git//example/pathA#branchA
    post_actions:
    - cluster_update
`,
			expectUnparsed: []Kapp{
				{
					Id:       "example1",
					State:    "present",
					manifest: nil,
					Templates: []Template{
						{
							"example/template1.tpl",
							"example/dest.txt",
						},
					},
					Sources: []acquirer.Source{
						{Id: "pathASpecial",
							Uri: "git@github.com:exampleA/repoA.git//example/pathA#branchA"},
						{Id: "sampleNameB",
							Uri: "git@github.com:exampleB/repoB.git//example/pathB#branchB"},
					},
				},
				{
					Id:       "example2",
					State:    "present",
					manifest: nil,
					Sources: []acquirer.Source{
						{
							Uri: "git@github.com:exampleA/repoA.git//example/pathA#branchA",
							Options: map[string]interface{}{
								"branch": "new-branch",
							},
						},
					},
					Vars: map[string]interface{}{
						"someVarA": "valueA",
						"someList": []interface{}{
							"val1",
							"val2",
						},
					},
				},
				{
					Id:       "example3",
					State:    "absent",
					manifest: nil,
					Sources: []acquirer.Source{
						{
							Uri: "git@github.com:exampleA/repoA.git//example/pathA#branchA",
						},
					},
					PostActions: []string{
						constants.TASK_ACTION_CLUSTER_UPDATE,
					},
				},
			},
			expectAcquirers: [][]acquirer.Acquirer{
				{
					// kapp1
					discardErr(acquirer.NewGitAcquirer(
						acquirer.Source{
							Id:  "pathASpecial",
							Uri: "git@github.com:exampleA/repoA.git//example/pathA#branchA",
						},
					)),
					discardErr(acquirer.NewGitAcquirer(
						acquirer.Source{
							Id:  "sampleNameB",
							Uri: "git@github.com:exampleB/repoB.git//example/pathB#branchB",
						})),
				},
				// kapp 2
				{
					discardErr(acquirer.NewGitAcquirer(
						acquirer.Source{
							Uri: "git@github.com:exampleA/repoA.git//example/pathA#branchA",
							Options: map[string]interface{}{
								"branch": "new-branch",
							},
						})),
				},
				// kapp3
				{
					discardErr(acquirer.NewGitAcquirer(
						acquirer.Source{
							Uri: "git@github.com:exampleA/repoA.git//example/pathA#branchA",
						},
					)),
				},
			},
			expectedError: false,
		},
	}

	for _, test := range tests {
		err := yaml.Unmarshal([]byte(test.input), &manifest)
		assert.Nil(t, err)

		if test.expectedError {
			assert.NotNil(t, err)
			assert.Nil(t, manifest.UnparsedKapps)
		} else {
			assert.Equal(t, test.expectUnparsed, manifest.UnparsedKapps, "unexpected conversion result for %s", test.name)
			assert.Nil(t, err)

			for i, parsedKapp := range manifest.ParsedKapps() {
				log.Logger.Infof("%#v", parsedKapp)
				acquirers, err := parsedKapp.Acquirers()
				assert.Nil(t, err)
				assert.Equal(t, test.expectAcquirers[i], acquirers, "unexpected acquirers for %s", test.name)
			}
		}
	}

	assert.NotEqual(t, manifest, Manifest{})
}

func TestManifestOverrides(t *testing.T) {

	// testing the correctness of this stack is handled in stack_test.go
	stackConfig, err := LoadStackConfig("large", "../../testdata/stacks.yaml")
	assert.Nil(t, err)
	assert.NotNil(t, stackConfig)

	expectedOverrides := map[string]interface{}{
		"state": "absent",
		"sources": map[interface{}]interface{}{
			"pathA": map[interface{}]interface{}{
				"options": map[interface{}]interface{}{
					"branch": "stable",
				},
			},
		},
		"vars": map[interface{}]interface{}{
			"stackVar": "setInOverrides",
			"sizeVar":  "mediumOverridden",
		},
	}

	actualOverrides, err := stackConfig.Manifests[0].ParsedKapps()[0].manifestOverrides()
	assert.Nil(t, err)

	assert.Equal(t, expectedOverrides, actualOverrides)
}

func TestManifestOverridesNil(t *testing.T) {

	// testing the correctness of this stack is handled in stack_test.go
	stackConfig, err := LoadStackConfig("large", "../../testdata/stacks.yaml")
	assert.Nil(t, err)
	assert.NotNil(t, stackConfig)

	actualOverrides, err := stackConfig.Manifests[1].ParsedKapps()[0].manifestOverrides()
	assert.Nil(t, err)
	assert.Nil(t, actualOverrides)
}

func TestApplyingManifestOverrides(t *testing.T) {

	// testing the correctness of this stack is handled in stack_test.go
	stackConfig, err := LoadStackConfig("large", "../../testdata/stacks.yaml")
	assert.Nil(t, err)
	assert.NotNil(t, stackConfig)

	// in the actual manifest, the state is set to present but it's overridden
	kappObj := stackConfig.Manifests[0].ParsedKapps()[0]
	assert.Equal(t, ABSENT_KEY, kappObj.State)
	assert.Equal(t, map[string]interface{}{
		"sizeVar":  "mediumOverridden",
		"stackVar": "setInOverrides",
		"colours": []interface{}{
			"red",
			"black",
		}}, kappObj.Vars)

	acquirers, err := kappObj.Acquirers()
	assert.Nil(t, err)
	assert.Equal(t, "git@github.com:sugarkube/kapps-A.git//some/pathA#stable", acquirers[0].Uri())
}
