package vars

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestInterfaceMapToStringMap(t *testing.T) {
	tests := []struct {
		name         string
		desc         string
		input        map[interface{}]interface{}
		expectValues map[string]string
	}{
		{
			name: "good_converstion",
			desc: "check converting expected input works",
			input: map[interface{}]interface{}{
				"testStr":   "hello",
				"testInt":   3,
				"testFloat": 1.11,
				"testBool":  true,
			},
			expectValues: map[string]string{
				"testStr":   "hello",
				"testInt":   "3",
				"testFloat": "1.11",
				"testBool":  "true",
			},
		},
	}

	for _, test := range tests {
		result := InterfaceMapToStringMap(test.input)
		assert.Equal(t, test.expectValues, result, "unexpected conversion result for %s", test.name)
	}
}
