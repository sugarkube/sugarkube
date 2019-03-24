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

package interfaces

import (
	"github.com/sugarkube/sugarkube/internal/pkg/provisioner"
	"github.com/sugarkube/sugarkube/internal/pkg/registry"
)

type IClusterStatus interface {
	IsOnline() bool
	SetIsOnline(bool)
	IsReady() bool
	SetIsReady(bool)
	StartedThisRun() bool
	SetStartedThisRun(bool)
	SleepBeforeReadyCheck() uint32
	SetSleepBeforeReadyCheck(uint32)
}

type IStackConfig interface {
	Name() string
	Provider() string
	Provisioner() string
	Account() string
	Region() string
	Profile() string
	Cluster() string
	OnlineTimeout() uint32
	ProviderVarsDirs() []string
	KappVarsDirs() []string
	TemplateDirs() []string
	Dir() string
}

type IStack interface {
	GetConfig() IStackConfig
	GetStatus() IClusterStatus
	GetProvisioner() provisioner.Provisioner
	GetRegistry() *registry.Registry
	TemplatedVars(installableObj IInstallable,
		installerVars map[string]interface{}) (map[string]interface{}, error)
}
