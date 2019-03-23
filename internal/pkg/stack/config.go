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
	"github.com/sugarkube/sugarkube/internal/pkg/log"
	"github.com/sugarkube/sugarkube/internal/pkg/structs"
	"os"
	"path/filepath"
)

// The populated config for a stack - all object addresses from the raw stack config have been
// replaced with actual instances
type StackConfig struct {
	rawConfig     structs.Stack
	providerVars  map[string]interface{}
	manifests     []*Manifest
	onlineTimeout uint32
	readyTimeout  uint32
}

// Returns the populated manifests
func (s StackConfig) Manifests() []*Manifest {
	return s.manifests
}

// Sets provider vars
func (s *StackConfig) SetProviderVars(vars map[string]interface{}) {
	s.providerVars = vars
}

// Gets provider vars
func (s *StackConfig) GetProviderVars() map[string]interface{} {
	return s.providerVars
}

func (s StackConfig) Name() string {
	return s.rawConfig.Name
}

func (s StackConfig) Provider() string {
	return s.rawConfig.Provider
}

func (s StackConfig) Provisioner() string {
	return s.rawConfig.Provisioner
}

func (s StackConfig) Account() string {
	return s.rawConfig.Account
}

func (s StackConfig) Profile() string {
	return s.rawConfig.Profile
}

func (s StackConfig) Cluster() string {
	return s.rawConfig.Cluster
}

func (s StackConfig) Region() string {
	return s.rawConfig.Region
}

// Validates that there aren't multiple manifests in the stack config with the
// same ID, which would break creating caches
func validateStackConfig(stackConfig *StackConfig) error {
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
func (s *StackConfig) Dir() string {
	if s.rawConfig.FilePath != "" {
		return filepath.Dir(s.rawConfig.FilePath)
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
		"name":        s.Name(),
		"filePath":    s.rawConfig.FilePath,
		"provider":    s.Provider(),
		"provisioner": s.Provisioner(),
		"account":     s.Account(),
		"region":      s.Region(),
		"profile":     s.Profile(),
		"cluster":     s.Cluster(),
	}
}
