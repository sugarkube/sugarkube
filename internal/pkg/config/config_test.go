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

package config

import (
	"github.com/stretchr/testify/assert"
	"github.com/sugarkube/sugarkube/internal/pkg/log"
	"path"
	"testing"
)

const testDir = "../../testdata"

func init() {
	log.ConfigureLogger("debug", false)
}

// Test that registry values override values when returning templated vars
func TestTemplatedVarsWithRegistry(t *testing.T) {
	configFile := path.Join(testDir, "test-sugarkube-conf.yaml")
	ViperConfig.SetConfigFile(configFile)

	err := Load(ViperConfig)
	assert.Nil(t, err)
}
