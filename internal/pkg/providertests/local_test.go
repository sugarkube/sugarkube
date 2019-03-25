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

package providertests

import (
	"github.com/stretchr/testify/assert"
	"github.com/sugarkube/sugarkube/internal/pkg/config"
	"github.com/sugarkube/sugarkube/internal/pkg/log"
	"github.com/sugarkube/sugarkube/internal/pkg/provider"
	"github.com/sugarkube/sugarkube/internal/pkg/stack"
	"github.com/sugarkube/sugarkube/internal/pkg/structs"
	"os"
	"path/filepath"
	"testing"
)

func init() {
	log.ConfigureLogger("debug", false)
}

func TestLocalVarsDirs(t *testing.T) {
	stackObj, err := stack.BuildStack("large", "../../testdata/stacks.yaml",
		&structs.Stack{}, &config.Config{}, os.Stdout)
	assert.Nil(t, err)

	absTestDir, err := filepath.Abs("../../testdata")
	assert.Nil(t, err)

	expected := []string{
		filepath.Join(absTestDir, "stacks/local/profiles/local/clusters/large/values.yaml"),
	}

	providerObj := &provider.LocalProvider{}
	actual, err := provider.FindVarsFiles(providerObj, stackObj.GetConfig())
	assert.Nil(t, err)

	assert.Equal(t, expected, actual, "Incorrect vars dirs returned")
}
