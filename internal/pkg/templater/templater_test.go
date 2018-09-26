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
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestRender(t *testing.T) {
	tests := []struct {
		name     string
		template string
		vars     map[string]interface{}
		expected string
	}{
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

	for _, test := range tests {
		result, err := Render(test.template, test.vars)
		assert.Nil(t, err)
		assert.Equal(t, test.expected, result,
			"Template rendering failed for %s", test.name)
	}
}
