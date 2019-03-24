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

package clustersot

import (
	"github.com/stretchr/testify/assert"
	"github.com/sugarkube/sugarkube/internal/pkg/interfaces"
	"github.com/sugarkube/sugarkube/internal/pkg/log"
	"github.com/sugarkube/sugarkube/internal/pkg/registry"
	"testing"
)

func init() {
	log.ConfigureLogger("debug", false)
}

type MockStack struct {
	stackConfig interfaces.IStackConfig
	status      interfaces.IClusterStatus
	provisioner interfaces.IProvisioner
	provider    interfaces.IProvider
	registry    *registry.Registry
}

func (m MockStack) GetConfig() interfaces.IStackConfig {
	return m.stackConfig
}

func (m MockStack) GetStatus() interfaces.IClusterStatus {
	return m.status
}

func (m MockStack) GetProvisioner() interfaces.IProvisioner {
	return m.provisioner
}

func (m MockStack) GetProvider() interfaces.IProvider {
	return m.provider
}

func (m MockStack) GetRegistry() *registry.Registry {
	return m.registry
}

func (m MockStack) TemplatedVars(installableObj interfaces.IInstallable,
	installerVars map[string]interface{}) (map[string]interface{}, error) {
	return nil, nil
}

func TestNewClusterSot(t *testing.T) {
	istack := MockStack{}

	actual, err := New(KubeCtl, istack)
	assert.Nil(t, err)
	assert.Equal(t, KubeCtlClusterSot{iStack: istack}, actual)
}
