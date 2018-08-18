package vars

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestGroupFilesWithDir(t *testing.T) {
	// we may want to mock filepath.Walk in future...
	result, err := GroupFiles("./testdata/value-merging/")
	assert.Nil(t, err)

	expected := map[string][]string{
		"values.yaml": {
			"testdata/value-merging/values.yaml",
			"testdata/value-merging/subdir1/values.yaml",
			"testdata/value-merging/subdir1/subdir2/values.yaml",
		},
	}

	assert.Equal(t, expected, result, "Failed to group files")
}

func TestGroupFilesWithFile(t *testing.T) {
	// we may want to mock filepath.Walk in future...
	result, err := GroupFiles("./testdata/value-merging/values.yaml")
	assert.Nil(t, err)

	expected := map[string][]string{
		"values.yaml": {
			"testdata/value-merging/values.yaml",
		},
	}

	assert.Equal(t, expected, result, "Failed to group files")
}
