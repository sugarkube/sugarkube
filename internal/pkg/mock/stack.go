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

package mock

import (
	"github.com/sugarkube/sugarkube/internal/pkg/interfaces"
	"github.com/sugarkube/sugarkube/internal/pkg/registry"
	"testing"
)

type Config struct {
	Name             string
	Provider         string
	Provisioner      string
	Account          string
	Region           string
	Profile          string
	Cluster          string
	OnlineTimeout    uint32
	ProviderVarsDirs []string
	Dir              string
}

func (c Config) GetName() string {
	return c.Name
}

func (c Config) GetProvider() string {
	return c.Provider
}

func (c Config) GetProvisioner() string {
	return c.Provisioner
}

func (c Config) GetAccount() string {
	return c.Account
}

func (c Config) GetRegion() string {
	return c.Region
}

func (c Config) GetProfile() string {
	return c.Profile
}

func (c Config) GetCluster() string {
	return c.Cluster
}

func (c Config) GetOnlineTimeout() uint32 {
	return c.OnlineTimeout
}

func (c Config) SetReadyTimeout(timeout uint32) {}

func (c Config) SetOnlineTimeout(timeout uint32) {}

func (c Config) GetProviderVarsDirs() []string {
	return c.ProviderVarsDirs
}

func (c Config) KappVarsDirs() []string {
	return nil
}

func (c Config) TemplateDirs() []string {
	return nil
}

func (c Config) GetDir() string {
	return c.Dir
}

func (c Config) Manifests() []interfaces.IManifest {
	return nil
}

func (c Config) GetIntrinsicData() map[string]string {
	return nil
}

func (c Config) GetProviderVars() map[string]interface{} {
	return nil
}

func (c Config) SetProviderVars(vars map[string]interface{}) {}

type MockStack struct {
	Config        interfaces.IStackConfig
	Provider      interfaces.IProvider
	TemplatedVars map[string]interface{}
	Registry      interfaces.IRegistry
}

func (m MockStack) GetConfig() interfaces.IStackConfig {
	return m.Config
}

func (m MockStack) GetStatus() interfaces.IClusterStatus {
	return nil
}

func (m MockStack) GetProvider() interfaces.IProvider {
	return m.Provider
}

func (m MockStack) GetProvisioner() interfaces.IProvisioner {
	return nil
}

func (m MockStack) GetRegistry() interfaces.IRegistry {
	return m.Registry
}

func (m MockStack) GetTemplatedVars(installableObj interfaces.IInstallable,
	installerVars map[string]interface{}) (map[string]interface{}, error) {
	return m.TemplatedVars, nil
}

func (m *MockStack) RefreshProviderVars() error {
	return nil
}

func (m *MockStack) LoadInstallables(workspaceDir string) error {
	return nil
}

func GetMockStackConfig(t *testing.T, dir string, name string, account string, provider string,
	provisioner string, profile string, cluster string, region string, providerVarsDirs []string) interfaces.IStackConfig {

	return Config{
		Name:             name,
		Account:          account,
		Provider:         provider,
		Provisioner:      provisioner,
		Profile:          profile,
		Cluster:          cluster,
		Region:           region,
		ProviderVarsDirs: providerVarsDirs,
		Dir:              dir,
	}
}

func GetMockStack(t *testing.T, dir string, name string, account string, provider string,
	provisioner string, profile string, cluster string, region string, providerVarsDirs []string) interfaces.IStack {

	registryObj := registry.New()

	config := GetMockStackConfig(t, dir, name, account, provider, provisioner, profile, cluster, region, providerVarsDirs)
	return &MockStack{
		Config:   config,
		Registry: registryObj,
	}
}
