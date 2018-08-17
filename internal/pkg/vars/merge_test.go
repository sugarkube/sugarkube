package vars

import (
	"github.com/stretchr/testify/assert"
	"path/filepath"
	"testing"
)

var (
	topPath  = "./testdata/values.yaml"
	subPath1 = "./testdata/subdir1/values.yaml"
	subPath2 = "./testdata/subdir1/subdir2/values.yaml"
)

func getAbsPath(t *testing.T, path string) string {
	absPath, err := filepath.Abs(path)
	if err != nil {
		t.Fatal(err)
	}

	return absPath
}

func TestMerge(t *testing.T) {
	topAbsPath := getAbsPath(t, topPath)
	sub1AbsPath := getAbsPath(t, subPath1)
	sub2AbsPath := getAbsPath(t, subPath2)

	tests := []struct {
		name         string
		desc         string
		paths        []string
		expectValues map[string]interface{}
	}{
		{
			name:  "no_merge",
			desc:  "check loading a single yaml file works",
			paths: []string{topAbsPath},
			expectValues: map[string]interface{}{
				"topString": "hello",
				"topBool":   true,
				"topInt":    999,
				"topFloat":  3.14,

				"topIntOvr": 5,

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
			name:  "check_overriding",
			desc:  "check merging a single level works",
			paths: []string{topAbsPath, sub1AbsPath},
			expectValues: map[string]interface{}{
				"topString": "hello",
				"topBool":   true,
				"topInt":    999,
				"topFloat":  3.14,

				"topIntOvr": 0,

				"subString": "subStr",
				"subBool":   false,
				"subInt":    11,
				"subFloat":  1.11,

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
		{
			name:  "check_overriding_two",
			desc:  "check merging two levels deep works",
			paths: []string{topAbsPath, sub1AbsPath, sub2AbsPath},
			expectValues: map[string]interface{}{
				"topString": "hello",
				"topBool":   true,
				"topInt":    999,
				"topFloat":  3.14,

				"topIntOvr": 8,

				"subString": "subStr",
				"subBool":   false,
				"subInt":    11,
				"subFloat":  1.11,

				"sub1": map[interface{}]interface{}{
					"subString": "subhello1",
					"subBool":   false,
					"subInt":    777,
					"subFloat":  6.22,

					"subStringOvr": "subgoodbye",
					"subBoolOvr":   false,
					"subIntOvr":    444,
					"subFloatOvr":  3.33,
				},

				"sub2Int": 10,
			},
		},
	}

	for _, test := range tests {
		result := *Merge(test.paths...)

		assert.Equal(t, test.expectValues, result, "unexpected merge result for %s", test.name)
	}
}
