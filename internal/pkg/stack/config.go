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
	"fmt"
	"github.com/pkg/errors"
	"github.com/sugarkube/sugarkube/internal/pkg/interfaces"
	"github.com/sugarkube/sugarkube/internal/pkg/log"
	"github.com/sugarkube/sugarkube/internal/pkg/structs"
	"os"
	"path/filepath"
)

// The populated config for a stack - all object addresses from the raw stack config have been
// replaced with actual instances
type StackConfig struct {
	stackFile     structs.StackFile
	providerVars  map[string]interface{}
	manifests     []interfaces.IManifest
	onlineTimeout uint32
	readyTimeout  uint32
}

// Returns the populated manifests
func (s StackConfig) Manifests() []interfaces.IManifest {
	return s.manifests
}

// Returns the configured list of provider vars dirs
func (s StackConfig) KappVarsDirs() []string {
	return s.stackFile.KappVarsDirs
}

// Sets the ready timeout
func (s *StackConfig) SetReadyTimeout(timeout uint32) {
	s.readyTimeout = timeout
}

// Sets the online timeout
func (s *StackConfig) SetOnlineTimeout(timeout uint32) {
	s.onlineTimeout = timeout
}

// Returns the configured list of template dirs
func (s StackConfig) TemplateDirs() []string {
	return s.stackFile.TemplateDirs
}

// Returns the configured list of provider vars dirs
func (s StackConfig) GetProviderVarsDirs() []string {
	return s.stackFile.ProviderVarsDirs
}

// Sets provider vars
func (s *StackConfig) SetProviderVars(vars map[string]interface{}) {
	s.providerVars = vars
}

// Gets provider vars
func (s StackConfig) GetProviderVars() map[string]interface{} {
	return s.providerVars
}

func (s StackConfig) GetName() string {
	return s.stackFile.Name
}

func (s StackConfig) GetProvider() string {
	return s.stackFile.Provider
}

func (s StackConfig) GetProvisioner() string {
	return s.stackFile.Provisioner
}

func (s StackConfig) GetAccount() string {
	return s.stackFile.Account
}

func (s StackConfig) GetProfile() string {
	return s.stackFile.Profile
}

func (s StackConfig) GetCluster() string {
	return s.stackFile.Cluster
}

func (s StackConfig) GetRegion() string {
	return s.stackFile.Region
}

func (s StackConfig) GetOnlineTimeout() uint32 {
	return s.onlineTimeout
}

// Validates that there aren't multiple manifests in the stack config with the
// same ID, which would break creating workspaces
func validateStackConfig(stackConfig interfaces.IStackConfig) error {
	ids := map[string]bool{}

	for _, manifest := range stackConfig.Manifests() {
		id := manifest.Id()

		if _, ok := ids[id]; ok {
			return errors.New(fmt.Sprintf("Multiple manifests exist with "+
				"the same id: %s", id))
		}
	}

	return nil
}

// Returns the directory the stack config was loaded from, or the current
// working directory. This can be used to build relative paths.
func (s *StackConfig) GetDir() string {
	if s.stackFile.FilePath != "" {
		return filepath.Dir(s.stackFile.FilePath)
	} else {
		// todo - remove this? It's probably unnecessary and will just introduce bugs/unexpected behaviour
		executable, err := os.Executable()
		if err != nil {
			log.Logger.Fatal("Failed to get the path of this binary.")
			panic(err)
		}

		return executable
	}
}

// Returns certain stack data that should be exposed as variables when running kapps
func (s *StackConfig) GetIntrinsicData() map[string]string {
	return map[string]string{
		"name":        s.GetName(),
		"filePath":    s.stackFile.FilePath,
		"provider":    s.GetProvider(),
		"provisioner": s.GetProvisioner(),
		"account":     s.GetAccount(),
		"region":      s.GetRegion(),
		"profile":     s.GetProfile(),
		"cluster":     s.GetCluster(),
	}
}
