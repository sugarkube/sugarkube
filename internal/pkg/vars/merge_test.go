package vars

import (
	"github.com/stretchr/testify/assert"
	"path/filepath"
	"testing"
)

var (
	topPath = "./testdata/values.yaml"
)

func TestMerge(t *testing.T) {
	topAbsPath, err := filepath.Abs(topPath)
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name         string
		desc         string
		paths        []string
		filterKeys   []string
		expectValues map[string]interface{}
	}{
		{
			name:       "check_name",
			desc:       "check for a known name in chart",
			paths:      []string{topAbsPath},
			filterKeys: []string{"topString", "topBool", "topInt", "topFloat"},
			expectValues: map[string]interface{}{
				"topString": "hello",
				"topBool":   true,
				"topInt":    999,
				"topFloat":  3.14,
			},
		},
	}

	for _, test := range tests {
		result := *Merge(test.paths...)

		filteredResult := map[string]interface{}{}

		for _, k := range test.filterKeys {
			filteredResult[k] = result[k]
		}

		assert.Equal(t, test.expectValues, filteredResult, "unexpected merge result")
	}
}
