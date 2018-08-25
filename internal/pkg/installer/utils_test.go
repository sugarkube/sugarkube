package installer

import (
	"github.com/stretchr/testify/assert"
	"path/filepath"
	"testing"
)

const testDir = "../../testdata"

// Test against testdata
func TestFindFilesByPattern(t *testing.T) {
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
				"../../testdata/manifests/manifest2.yaml",
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
				"../../testdata/manifests/manifest1.yaml",
				"../../testdata/manifests/manifest2.yaml",
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
				"../../testdata/value-merging/values.yaml",
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
				"../../testdata/value-merging/subdir1/subdir2/values.yaml",
				"../../testdata/value-merging/subdir1/values.yaml",
			},
		},
		{
			name: "good_symlinks",
			desc: "test that symlinks are followed",
			// todo - generate a proper set of test dirs
			startDir:       "../../../test-cache/web/wordpress",
			pattern:        "values.yaml",
			recursive:      true,
			preferSymlinks: true,
			expectValues: []string{
				"../../../test-cache/web/wordpress/wordpress/values.yaml",
			},
		},
	}

	for _, test := range tests {
		result, err := findFilesByPattern(test.startDir, test.pattern,
			test.recursive, test.preferSymlinks)
		assert.Nil(t, err)
		assert.Equal(t, test.expectValues, result, "unexpected files returned for %s", test.name)
	}
}
