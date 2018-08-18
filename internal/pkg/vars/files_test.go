package vars

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestGroupFilesWithDir(t *testing.T) {
	// we may want to mock filepath.Walk in future...
	result := GroupFiles("./testdata")

	expected := map[string][]string{
		"values.yaml": {
			"testdata/values/values.yaml",
			"testdata/values/subdir1/values.yaml",
			"testdata/values/subdir1/subdir2/values.yaml",
		},
		"stacks.yaml": {
			"testdata/stacks.yaml",
		},
	}

	assert.Equal(t, expected, result, "Failed to group files")
}

func TestGroupFilesWithFile(t *testing.T) {
	// we may want to mock filepath.Walk in future...
	result := GroupFiles("./testdata/values/values.yaml")

	expected := map[string][]string{
		"values.yaml": {
			"testdata/values/values.yaml",
		},
	}

	assert.Equal(t, expected, result, "Failed to group files")
}
