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

package utils

import (
	"github.com/stretchr/testify/assert"
	"path/filepath"
	"testing"
)

const testDir = "../../testdata"

// Test against testdata
func TestFindFilesByPattern(t *testing.T) {

	absTestDir, err := filepath.Abs(testDir)
	assert.Nil(t, err)

	tests := []struct {
		name           string
		desc           string
		startDir       string
		pattern        string
		recursive      bool
		preferSymlinks bool
		expectValues   []string
	}{
		{
			name:           "good_no_pattern",
			desc:           "test that files are found when no regex characters are used in the pattern",
			startDir:       testDir,
			pattern:        "manifest2.yaml",
			recursive:      true,
			preferSymlinks: true,
			expectValues: []string{
				filepath.Join(absTestDir, "manifests/manifest2.yaml"),
			},
		},
		{
			name:           "good_simple_pattern",
			desc:           "test that files are found when a regex pattern is used",
			startDir:       testDir,
			pattern:        "manifest\\d.yaml",
			recursive:      true,
			preferSymlinks: true,
			expectValues: []string{
				filepath.Join(absTestDir, "manifests/manifest1.yaml"),
				filepath.Join(absTestDir, "manifests/manifest2.yaml"),
			},
		},
		{
			name:           "good_no_recursion",
			desc:           "test that recursion can be disabled",
			startDir:       filepath.Join(testDir, "value-merging"),
			pattern:        "values.yaml",
			recursive:      false,
			preferSymlinks: true,
			expectValues: []string{
				filepath.Join(absTestDir, "value-merging/values.yaml"),
			},
		},
		{
			name:           "good_recursion",
			desc:           "test that recursion paths are returned from multiple directories",
			startDir:       filepath.Join(testDir, "value-merging", "subdir1"),
			pattern:        "values.yaml",
			recursive:      true,
			preferSymlinks: true,
			expectValues: []string{
				filepath.Join(absTestDir, "value-merging/subdir1/subdir2/values.yaml"),
				filepath.Join(absTestDir, "value-merging/subdir1/values.yaml"),
			},
		},
		// todo - generate a proper set of test dirs and re-enable this
		//{
		//	name: "good_symlinks",
		//	desc: "test that symlinks are followed",
		//	startDir:       "../../../test-cache/web/wordpress",
		//	pattern:        "values.yaml",
		//	recursive:      true,
		//	preferSymlinks: true,
		//	expectValues: []string{
		//		"../../../test-cache/web/wordpress/wordpress/values.yaml",
		//	},
		//},
	}

	for _, test := range tests {
		result, err := FindFilesByPattern(test.startDir, test.pattern,
			test.recursive, test.preferSymlinks)
		assert.Nil(t, err)
		assert.Equal(t, test.expectValues, result, "unexpected files returned for %s", test.name)
	}
}
