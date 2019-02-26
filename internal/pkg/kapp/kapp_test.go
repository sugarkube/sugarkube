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
    sampleNameB:
      - uri: git@github.com:exampleB/repoB.git//example/pathB#branchB

  - id: example2
    state: present
    sources:
    - uri: git@github.com:exampleA/repoA.git//example/pathA#branchA
    vars:
      someVarA: valueA
      someList:
      - val1
      - val2

  - id: example3
    state: absent
    sources:
    - uri: git@github.com:exampleA/repoA.git//example/pathA#branchA
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
					},
				},
				{
					Id:       "example2",
					State:    "present",
					manifest: nil,
					Sources: []acquirer.Source{
						{Uri: "git@github.com:exampleA/repoA.git//example/pathA#branchA"},
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
				},
			},
			expectAcquirers: [][]acquirer.Acquirer{
				{
					// kapp1
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
				// kapp 2
				{
					discardErr(acquirer.NewGitAcquirer(
						"pathA",
						"git@github.com:exampleA/repoA.git",
						"branchA",
						"example/pathA",
						"")),
				},
				// kapp3
				{
					discardErr(acquirer.NewGitAcquirer(
						"pathA",
						"git@github.com:exampleA/repoA.git",
						"branchA",
						"example/pathA",
						"")),
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
				acquirers, err := parsedKapp.Acquirers()
				assert.Nil(t, err)
				assert.Equal(t, test.expectAcquirers[i], acquirers, "unexpected acquirers for %s", test.name)
			}
		}
	}

	assert.NotEqual(t, manifest, Manifest{})
}
