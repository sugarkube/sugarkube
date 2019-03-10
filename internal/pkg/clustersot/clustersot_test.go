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
	"github.com/sugarkube/sugarkube/internal/pkg/kapp"
	"github.com/sugarkube/sugarkube/internal/pkg/log"
	"testing"
)

func init() {
	log.ConfigureLogger("debug", false)
}

type MockStack struct {
	stackConfig *kapp.StackConfig
	status      interfaces.IClusterStatus
}

func (m MockStack) GetConfig() *kapp.StackConfig {
	return m.stackConfig
}

func (m MockStack) GetStatus() interfaces.IClusterStatus {
	return m.status
}

func TestNewClusterSot(t *testing.T) {
	istack := MockStack{}

	actual, err := NewClusterSot(KUBECTL, istack)
	assert.Nil(t, err)
	assert.Equal(t, KubeCtlClusterSot{stack: istack}, actual)
}
