/*
 * Copyright 2018 The Sugarkube Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

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
			input:    newManifest("example/manifest.yaml"),
			expected: "manifest",
		},
	}

	for _, test := range tests {
		SetManifestDefaults(&test.input)
		assert.Equal(t, test.expected, test.input.Id)
	}
}
