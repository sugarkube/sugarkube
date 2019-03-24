/*
 * Copyright 2019 The Sugarkube Authors
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

package stack

import (
	"github.com/stretchr/testify/assert"
	"github.com/sugarkube/sugarkube/internal/pkg/config"
	"github.com/sugarkube/sugarkube/internal/pkg/installable"
	"github.com/sugarkube/sugarkube/internal/pkg/interfaces"
	"github.com/sugarkube/sugarkube/internal/pkg/structs"
	"os"
	"testing"
)

// Helper to get acquirers in a single-valued context
func discardErrInstallable(installable interfaces.IInstallable, err error) interfaces.IInstallable {
	if err != nil {
		panic(err)
	}

	return installable
}

func TestValidateManifest(t *testing.T) {
	testManifestId := "testManifest"
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
				installables: []interfaces.IInstallable{
					discardErrInstallable(installable.New(testManifestId, structs.KappDescriptor{Id: "example1"})),
					discardErrInstallable(installable.New(testManifestId, structs.KappDescriptor{Id: "example2"})),
				},
			},
		},
		{
			name: "error_multiple_kapps_same_id",
			desc: "error when kapp IDs aren't unique",
			input: Manifest{
				installables: []interfaces.IInstallable{
					discardErrInstallable(installable.New(testManifestId, structs.KappDescriptor{Id: "example1"})),
					discardErrInstallable(installable.New(testManifestId, structs.KappDescriptor{Id: "example2"})),
					discardErrInstallable(installable.New(testManifestId, structs.KappDescriptor{Id: "example1"})),
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
		input    interfaces.IManifest
		expected string
	}{
		{
			name: "good",
			desc: "default manifest IDs should be the URI basename minus extension",
			input: &Manifest{
				descriptor: structs.ManifestDescriptor{Id: "", Uri: "example/manifest.yaml"},
			},
			expected: "manifest",
		},
	}

	for _, test := range tests {
		assert.Equal(t, test.expected, test.input.Id())
	}
}

func TestSelectKapps(t *testing.T) {

	// testing the correctness of this stack is handled in stack_test.go
	stackConfig, err := BuildStack("kops", "../../testdata/stacks.yaml",
		&structs.Stack{}, &config.Config{}, os.Stdout)
	assert.Nil(t, err)
	assert.NotNil(t, stackConfig)

	includeSelector := []string{
		"exampleManifest2:kappA",
		"manifest1:kappA",
		"exampleManifest2:kappB",
	}

	var excludeSelector []string

	expectedKappIds := []string{
		"manifest1:kappA",
		"exampleManifest2:kappB",
		"exampleManifest2:kappA",
	}

	selectedKapps, err := SelectInstallables(stackConfig.GetConfig().Manifests(), includeSelector, excludeSelector)
	assert.Nil(t, err)

	for i := 0; i < len(expectedKappIds); i++ {
		assert.Equal(t, expectedKappIds[i], selectedKapps[i].FullyQualifiedId())
	}
}

func TestSelectKappsExclusions(t *testing.T) {

	// testing the correctness of this stack is handled in stack_test.go
	stack, err := BuildStack("kops", "../../testdata/stacks.yaml",
		&structs.Stack{}, &config.Config{}, os.Stdout)
	assert.Nil(t, err)
	assert.NotNil(t, stack)

	includeSelector := []string{
		"exampleManifest2:*",
		"manifest1:kappA",
	}

	excludeSelector := []string{
		"exampleManifest2:kappA",
	}

	expectedKappIds := []string{
		"manifest1:kappA",
		"exampleManifest2:kappC",
		"exampleManifest2:kappB",
		"exampleManifest2:kappD",
	}

	selectedKapps, err := SelectInstallables(stack.GetConfig().Manifests(), includeSelector, excludeSelector)
	assert.Nil(t, err)

	for i := 0; i < len(expectedKappIds); i++ {
		assert.Equal(t, expectedKappIds[i], selectedKapps[i].FullyQualifiedId())
	}
}
