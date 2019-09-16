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

package provider

import (
	"github.com/stretchr/testify/assert"
	"github.com/sugarkube/sugarkube/internal/pkg/log"
	"github.com/sugarkube/sugarkube/internal/pkg/mock"
	"os"
	"path/filepath"
	"testing"
)

func init() {
	log.ConfigureLogger("debug", false, os.Stderr)
}

const testDir = "../../testdata"

func TestLocalVarsDirs(t *testing.T) {
	stackConfig := mock.GetMockStackConfig(t, testDir, "large", "", "local",
		"minikube", "local", "large", "fake-region", []string{"./stacks/"})

	assert.Equal(t, "local", stackConfig.GetProvider())
	assert.Equal(t, []string{"./stacks/"}, stackConfig.GetProviderVarsDirs())

	absTestDir, err := filepath.Abs(testDir)
	assert.Nil(t, err)

	expected := []string{
		filepath.Join(absTestDir, "stacks/local/profiles/local/clusters/large/values.yaml"),
	}

	providerObj := &LocalProvider{}
	actual, err := findVarsFiles(providerObj, stackConfig)
	assert.Nil(t, err)

	assert.Equal(t, expected, actual, "Incorrect vars dirs returned")
}
