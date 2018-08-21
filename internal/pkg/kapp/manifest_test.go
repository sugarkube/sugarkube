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
					{Id: "example1"},
					{Id: "example2"},
				},
			},
		},
		{
			name: "error_multiple_kapps_same_id",
			desc: "error when kapp IDs aren't unique",
			input: Manifest{
				Kapps: []Kapp{
					{Id: "example1"},
					{Id: "example2"},
					{Id: "example1"},
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

func TestSetManifestDefaults(t *testing.T) {
	tests := []struct {
		name     string
		desc     string
		input    Manifest
		expected string
	}{
		{
			name:     "good",
			desc:     "default manifest IDs should be the URI basename minus extension",
			input:    NewManifest("example/manifest.yaml"),
			expected: "manifest",
		},
	}

	for _, test := range tests {
		SetManifestDefaults(&test.input)
		assert.Equal(t, test.expected, test.input.Id)
	}
}
