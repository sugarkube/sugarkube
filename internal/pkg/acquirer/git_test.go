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

package acquirer

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

// Helper to get acquirers in a single-valued context
func discardErr(acquirer Acquirer, err error) Acquirer {
	if err != nil {
		panic(err)
	}

	return acquirer
}

func TestId(t *testing.T) {
	// the URI is invalid. It should cause an error
	invalidUriAcquirer, err := NewGitAcquirer(
		"",
		"git@github.com:helm:thing/charts.git",
		"master",
		"stable/wordpress",
		"")
	assert.Nil(t, invalidUriAcquirer)
	assert.NotNil(t, err)

	tests := []struct {
		name         string
		desc         string
		input        Acquirer
		expectValues string
		expectError  bool
	}{
		{
			name: "good",
			desc: "check IDs are generated with expected input",
			input: discardErr(NewGitAcquirer(
				"",
				"git@github.com:helm/charts.git",
				"master",
				"stable/wordpress",
				"")),
			expectValues: "helm-charts-wordpress",
		},
		{
			name: "good_path_leading_trailing_slash",
			desc: "check leading/trailing slashes on paths don't affect IDs",
			input: discardErr(NewGitAcquirer(
				"",
				"git@github.com:helm/charts.git",
				"master",
				"/stable/wordpress/",
				"")),
			expectValues: "helm-charts-wordpress",
		},
		{
			name: "good_name_in_id",
			desc: "check explicit names are put into IDs",
			input: discardErr(NewGitAcquirer(
				"site1-values",
				"git@github.com:sugarkube/sugarkube.git",
				"master",
				"examples/values/wordpress/site1/",
				"")),
			expectValues: "sugarkube-sugarkube-site1-values",
		},
	}

	for _, test := range tests {
		result, err := test.input.FullyQualifiedId()

		if test.expectError {
			assert.NotNil(t, err)
			assert.Empty(t, result)
		} else {
			assert.Nil(t, err)
			assert.Equal(t, test.expectValues, result, "IDs don't match")
		}
	}
}
