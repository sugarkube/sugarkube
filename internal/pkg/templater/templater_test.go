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

package templater

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

type TemplateTest struct {
	name     string
	template string
	vars     map[string]interface{}
	expected string
}

func getFixture() []TemplateTest {
	return []TemplateTest{
		{
			name:     "no-vars",
			template: `{{ "hello!" | upper | repeat 5 }}`,
			expected: "HELLO!HELLO!HELLO!HELLO!HELLO!",
			vars:     nil,
		},
		{
			name:     "simple-vars",
			template: `hello {{ .place | upper }}!`,
			expected: "hello WORLD!",
			vars: map[string]interface{}{
				"place": "world",
			},
		},
		{
			name:     "nested-vars",
			template: `hello {{ .places.planet | upper }}!`,
			expected: "hello EARTH!",
			vars: map[string]interface{}{
				"places": map[string]interface{}{
					"planet": "earth",
				},
			},
		},
	}
}

func TestRenderTemplate(t *testing.T) {
	tests := getFixture()

	for _, test := range tests {
		result, err := renderTemplate(test.template, test.vars)
		assert.Nil(t, err)
		assert.Equal(t, test.expected, result,
			"Template rendering failed for %s", test.name)
	}
}

func TestTemplateFile(t *testing.T) {
	tests := getFixture()

	inputTempDir, err := ioutil.TempDir("", "inputTpls-")
	assert.Nil(t, err)

	outputTempDir, err := ioutil.TempDir("", "outputTpls-")
	assert.Nil(t, err)

	for i, test := range tests {
		inputTemplatePath := filepath.Join(inputTempDir,
			fmt.Sprintf("test-%d.tpl", i))
		outputTemplatePath := filepath.Join(outputTempDir,
			fmt.Sprintf("test-%d.txt", i))

		err = ioutil.WriteFile(inputTemplatePath, []byte(test.template), 0644)
		assert.Nil(t, err)

		outFile, err := os.Create(outputTemplatePath)
		assert.Nil(t, err)
		defer outFile.Close()

		err = TemplateFile(inputTemplatePath, outFile, test.vars)
		assert.Nil(t, err)

		result, err := ioutil.ReadFile(outputTemplatePath)
		assert.Nil(t, err)

		assert.Equal(t, test.expected, string(result[:]),
			"Template rendering failed for %s", test.name)
	}
}
