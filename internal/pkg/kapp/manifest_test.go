package kapp

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestValidateManifest(t *testing.T) {
	tests := []struct {
		name          string
		desc          string
		input         Manifest
		expectedError bool
	}{
		{
			name: "good",
			desc: "kapp IDs should be unique",
			input: Manifest{
				Kapps: []Kapp{
					{id: "example1"},
					{id: "example2"},
				},
			},
		},
		{
			name: "error_multiple_kapps_same_id",
			desc: "error when kapp IDs aren't unique",
			input: Manifest{
				Kapps: []Kapp{
					{id: "example1"},
					{id: "example2"},
					{id: "example1"},
				},
			},
		},
	}

	for _, test := range tests {
		err := ValidateManifest(&test.input)
		if test.expectedError {
			assert.NotNil(t, err)
		} else {
			assert.Nil(t, err)
		}
	}
}
