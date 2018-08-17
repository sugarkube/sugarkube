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
			"testdata/values.yaml",
			"testdata/subdir1/values.yaml",
			"testdata/subdir1/subdir2/values.yaml",
		},
	}

	assert.Equal(t, expected, result, "Failed to group files")
}

func TestGroupFilesWithFile(t *testing.T) {
	// we may want to mock filepath.Walk in future...
	result := GroupFiles("./testdata/values.yaml")

	expected := map[string][]string{
		"values.yaml": {
			"testdata/values.yaml",
		},
	}

	assert.Equal(t, expected, result, "Failed to group files")
}
