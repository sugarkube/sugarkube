package vars

import (
	"github.com/stretchr/testify/assert"
	"path/filepath"
	"testing"
)

var (
	topPath  = "./testdata/values.yaml"
	subPath1 = "./testdata/subdir1/values.yaml"
)

func TestMerge(t *testing.T) {
	topAbsPath, err := filepath.Abs(topPath)
	if err != nil {
		t.Fatal(err)
	}

	sub1AbsPath, err := filepath.Abs(subPath1)
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
			name:       "no_merge",
			desc:       "check loading a single yaml file works",
			paths:      []string{topAbsPath},
			filterKeys: []string{"topString", "topBool", "topInt", "topFloat", "sub1"},
			expectValues: map[string]interface{}{
				"topString": "hello",
				"topBool":   true,
				"topInt":    999,
				"topFloat":  3.14,

				"sub1": map[interface{}]interface{}{
					"subString": "subhello1",
					"subBool":   true,
					"subInt":    777,
					"subFloat":  6.22,

					"subStringOvr": "subhello2",
					"subBoolOvr":   true,
					"subIntOvr":    777,
					"subFloatOvr":  6.22,
				},
			},
		},
		{
			name:       "check_overriding",
			desc:       "check overriding a single subkey works",
			paths:      []string{topAbsPath, sub1AbsPath},
			filterKeys: []string{"sub1"},
			expectValues: map[string]interface{}{
				"sub1": map[interface{}]interface{}{
					"subString": "subhello1",
					"subBool":   true,
					"subInt":    777,
					"subFloat":  6.22,

					"subStringOvr": "subgoodbye",
					"subBoolOvr":   false,
					"subIntOvr":    444,
					"subFloatOvr":  3.33,
				},
			},
		},
	}

	for _, test := range tests {
		result := *Merge(test.paths...)

		filteredResult := map[string]interface{}{}

		for _, k := range test.filterKeys {
			filteredResult[k] = result[k]
		}

		assert.Equal(t, test.expectValues, filteredResult, "unexpected merge result for %s", test.name)
	}
}
