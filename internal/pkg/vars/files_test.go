package vars

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestGroupFiles(t *testing.T) {
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
