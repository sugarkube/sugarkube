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
	GetName() string
	GetProvider() string
	GetProvisioner() string
	GetAccount() string
	GetRegion() string
	GetProfile() string
	GetCluster() string
	GetOnlineTimeout() uint32
	SetReadyTimeout(timeout uint32)
	SetOnlineTimeout(timeout uint32)
	GetProviderVarsDirs() []string
	KappVarsDirs() []string
	TemplateDirs() []string
	GetDir() string
	Manifests() []IManifest
	GetIntrinsicData() map[string]string
	GetProviderVars() map[string]interface{}
	SetProviderVars(vars map[string]interface{})
}

type IStack interface {
	GetConfig() IStackConfig
	GetStatus() IClusterStatus
	GetProvider() IProvider
	GetProvisioner() IProvisioner
	GetRegistry() IRegistry
	GetTemplatedVars(installableObj IInstallable,
		installerVars map[string]interface{}) (map[string]interface{}, error)
	RefreshProviderVars() error
	LoadInstallables(workspaceDir string) error
}
