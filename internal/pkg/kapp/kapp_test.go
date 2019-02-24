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
	"github.com/sugarkube/sugarkube/internal/pkg/log"
	"gopkg.in/yaml.v2"
	"testing"
)

func init() {
	log.ConfigureLogger("debug", false)
}

func TestParseManifestYaml(t *testing.T) {
	manifest := Manifest{
		Uri: "fake/uri",
		Id:  "test-manifest",
	}

	tests := []struct {
		name                 string
		desc                 string
		input                string
		inputShouldBePresent bool
		expectValues         []Kapp
		expectedError        bool
	}{
		{
			name: "good_parse",
			desc: "check parsing acceptable input works",
			input: `
present:
  - id: example1
    templates:        
      - source: example/template1.tpl
        dest: example/dest.txt
    sources:
    - uri: git@github.com:exampleA/repoA.git
      branch: branchA
      path: example/pathA
    - uri: git@github.com:exampleB/repoB.git
      branch: branchB
      path: example/pathB
      name: sampleNameB

  - id: example2
    sources:
    - uri: git@github.com:exampleA/repoA.git
      branch: branchA
      path: example/pathA
    vars:
      someVarA: valueA
      someList:
      - val1
      - val2

absent:
  - id: example3
    sources:
    - uri: git@github.com:exampleA/repoA.git
      branch: branchA
      path: example/pathA
`,
			expectValues: []Kapp{
				{
					Id:              "example1",
					ShouldBePresent: true,
					manifest:        &manifest,
					Templates: []Template{
						{
							"example/template1.tpl",
							"example/dest.txt",
						},
					},
					Sources: []acquirer.Acquirer{
						discardErr(acquirer.NewGitAcquirer(
							"pathA",
							"git@github.com:exampleA/repoA.git",
							"branchA",
							"example/pathA",
							"")),
						discardErr(acquirer.NewGitAcquirer(
							"sampleNameB",
							"git@github.com:exampleB/repoB.git",
							"branchB",
							"example/pathB",
							"")),
					},
				},
				{
					Id:              "example2",
					ShouldBePresent: true,
					manifest:        &manifest,
					Sources: []acquirer.Acquirer{
						discardErr(acquirer.NewGitAcquirer(
							"pathA",
							"git@github.com:exampleA/repoA.git",
							"branchA",
							"example/pathA",
							"")),
					},
					vars: map[string]interface{}{
						"someVarA": "valueA",
						"someList": []interface{}{
							"val1",
							"val2",
						},
					},
				},
				{
					Id:              "example3",
					ShouldBePresent: false, // should be absent
					manifest:        &manifest,
					Sources: []acquirer.Acquirer{
						discardErr(acquirer.NewGitAcquirer(
							"pathA",
							"git@github.com:exampleA/repoA.git",
							"branchA",
							"example/pathA",
							"")),
					},
				},
			},
			expectedError: false,
		},
	}

	for _, test := range tests {
		inputYaml := map[string]interface{}{}
		err := yaml.Unmarshal([]byte(test.input), inputYaml)
		assert.Nil(t, err)

		result, err := parseManifestYaml(&manifest, inputYaml)
		if test.expectedError {
			assert.NotNil(t, err)
			assert.Nil(t, result)
		} else {
			assert.Equal(t, test.expectValues, result, "unexpected conversion result for %s", test.name)
			assert.Nil(t, err)
		}
	}

	assert.NotEqual(t, manifest, Manifest{})
}
