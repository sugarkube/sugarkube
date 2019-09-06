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
	"bytes"
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/sugarkube/sugarkube/internal/pkg/log"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

const testDir = "../../testdata"

type TemplateTest struct {
	name     string
	template string
	vars     map[string]interface{}
	expected string
}

func init() {
	log.ConfigureLogger("debug", false, os.Stderr)
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
		result, err := RenderTemplate(test.template, test.vars)
		assert.Nil(t, err)
		assert.Equal(t, test.expected, result,
			"Template rendering failed for %s", test.name)
	}
}

func TestTemplateFile(t *testing.T) {
	tests := getFixture()

	inputTempDir, err := ioutil.TempDir("", "inputTpls-")
	assert.Nil(t, err)

	for i, test := range tests {
		inputTemplatePath := filepath.Join(inputTempDir,
			fmt.Sprintf("test-%d.tpl", i))

		err = ioutil.WriteFile(inputTemplatePath, []byte(test.template), 0644)
		assert.Nil(t, err)

		var outBuf bytes.Buffer
		err = TemplateFile(inputTemplatePath, &outBuf, test.vars)
		assert.Nil(t, err)

		assert.Equal(t, test.expected, outBuf.String(),
			"Template rendering failed for %s", test.name)
	}
}

func TestCustomFunctions(t *testing.T) {

	absTestDir, err := filepath.Abs(testDir)
	assert.Nil(t, err)
	sampleWorkspaceRoot := filepath.Join(absTestDir, "sample-workspace", "sample-manifest", "sample-kapp")

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:  "test findFiles",
			input: `-f {{ listString "/values.yaml$" | findFiles .kapp.cacheRoot | uniq | join " " }} {{ mapPrintF "/values-%s.yaml$" .sugarkube.defaultVars | findFiles .kapp.cacheRoot | mapPrintF "-f %s" | uniq | join " " }}`,
			expected: fmt.Sprintf("-f %s/sample-chart/values.yaml -f %s/sample-chart/values-dev.yaml "+
				"-f %s/sample-chart/values-dev1.yaml", sampleWorkspaceRoot, sampleWorkspaceRoot, sampleWorkspaceRoot),
		},
	}

	for _, test := range tests {
		output, err := RenderTemplate(test.input, map[string]interface{}{
			"sugarkube": map[string]interface{}{
				// duplicate matches should be stripped out (note "dev" would match twice)
				"defaultVars": []string{"missing-provider", "dev", "dev", "dev1", "missing-region"},
			},
			"kapp": map[string]interface{}{
				"cacheRoot": sampleWorkspaceRoot,
			},
		})
		assert.Nil(t, err)
		assert.Equal(t, test.expected, output)
	}
}
