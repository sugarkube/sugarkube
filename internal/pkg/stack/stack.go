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
	"github.com/sugarkube/sugarkube/internal/pkg/clustersot"
	"github.com/sugarkube/sugarkube/internal/pkg/convert"
	"github.com/sugarkube/sugarkube/internal/pkg/interfaces"
	"github.com/sugarkube/sugarkube/internal/pkg/log"
	"github.com/sugarkube/sugarkube/internal/pkg/provider"
	"github.com/sugarkube/sugarkube/internal/pkg/provisioner"
	"github.com/sugarkube/sugarkube/internal/pkg/templater"
	"github.com/sugarkube/sugarkube/internal/pkg/vars"
	"gopkg.in/yaml.v2"
	"os"
	"path/filepath"
)

// Top-level struct that holds references to instantiations of other objects
// we need to pass around. This is in its own package to avoid circular
// dependencies.
type Stack struct {
	config      interfaces.IStackConfig
	provider    interfaces.IProvider
	provisioner interfaces.IProvisioner
	status      *ClusterStatus
	registry    interfaces.IRegistry
}

// Creates a new Stack
func newStack(config interfaces.IStackConfig, provider interfaces.IProvider, registry interfaces.IRegistry) (interfaces.IStack, error) {

	stack := &Stack{
		config:      config,
		provider:    provider,
		provisioner: nil,
		status: &ClusterStatus{
			isOnline:              false,
			isReady:               false,
			sleepBeforeReadyCheck: 0,
			startedThisRun:        false,
		},
		registry: registry,
	}

	err := stack.RefreshProviderVars()
	if err != nil {
		return nil, errors.WithStack(err)
	}

	clusterSot, err := clustersot.New(clustersot.KubeCtl, stack)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	provisionerImpl, err := provisioner.New(stack.config.GetProvisioner(), stack, clusterSot)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	stack.provisioner = provisionerImpl

	return stack, nil
}

func (s Stack) GetConfig() interfaces.IStackConfig {
	return s.config
}

func (s Stack) GetStatus() interfaces.IClusterStatus {
	return s.status
}

func (s Stack) GetProvisioner() interfaces.IProvisioner {
	return s.provisioner
}

func (s Stack) GetProvider() interfaces.IProvider {
	return s.provider
}

func (s Stack) GetRegistry() interfaces.IRegistry {
	return s.registry
}

// Merges and templates vars from all configured sources. If an installable instance
// is given, data specific to it will be included in the returned map,
// otherwise only stack-specific variables will be returned.
func (s *Stack) GetTemplatedVars(installableObj interfaces.IInstallable,
	installerVars map[string]interface{}) (map[string]interface{}, error) {

	stackConfig := s.config

	// build an array of config fragments that should all be merged together,
	// with later values overriding earlier ones.
	configFragments := make([]map[string]interface{}, 0)

	stackIntrinsicData := stackConfig.GetIntrinsicData()
	// convert the map to the appropriate type and namespace it
	configFragments = append(configFragments, map[string]interface{}{
		"stack": convert.MapStringStringToMapStringInterface(stackIntrinsicData),
	})

	// store additional runtime values under the "sugarkube" key
	installerVars["defaultVars"] = []string{
		stackConfig.GetProvider(),
		stackConfig.GetProvisioner(),
		stackConfig.GetAccount(), // may be blank depending on the provider
		stackConfig.GetProfile(),
		stackConfig.GetCluster(),
		stackConfig.GetRegion(), // may be blank depending on the provider
	}

	configFragments = append(configFragments, map[string]interface{}{
		"sugarkube": installerVars,
	})

	configFragments = append(configFragments, stackConfig.GetProviderVars())

	// merge in values from the registry
	log.Logger.Tracef("Merging stack vars with global registry: %v", s.registry)
	configFragments = append(configFragments, s.registry.AsMap())

	if installableObj != nil {
		installableVars, err := installableObj.Vars(s)
		if err != nil {
			return nil, errors.WithStack(err)
		}

		configFragments = append(configFragments, installableVars)
	}

	mergedVars := map[string]interface{}{}
	err := vars.MergeFragments(&mergedVars, configFragments...)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	templatedVars, err := templater.IterativelyTemplate(mergedVars)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	yamlData, err := yaml.Marshal(&templatedVars)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	log.Logger.Tracef("Vars after merging and templating:\n%s", yamlData)

	return templatedVars, nil
}

// Reload provider vars
func (s *Stack) RefreshProviderVars() error {
	providerVars, err := provider.GetVarsFromFiles(s.GetProvider(), s.GetConfig())
	if err != nil {
		log.Logger.Warn("Error loading provider variables")
		return errors.WithStack(err)
	}
	log.Logger.Tracef("Provider loaded vars: %#v", providerVars)

	if len(providerVars) == 0 {
		log.Logger.Error("No values loaded for provider")
		return errors.New(fmt.Sprintf("Failed to load variables for provider %s",
			s.GetProvider().GetName()))
	}

	s.GetConfig().SetProviderVars(providerVars)
	return nil
}

// Loads the configs for all installables
func (s *Stack) LoadInstallables(workspaceDir string) error {
	installables := make([]interfaces.IInstallable, 0)

	// load configs for all installables so we can build a DAG and load outputs
	// for dependencies, etc.
	for _, manifest := range s.GetConfig().Manifests() {
		for _, installable := range manifest.Installables() {
			installables = append(installables, installable)
		}
	}

	// set the workspace dir on each installable if it's non-empty
	if workspaceDir != "" {
		absWorkspaceDir, err := filepath.Abs(workspaceDir)
		if err != nil {
			return errors.WithStack(err)
		}

		if _, err := os.Stat(absWorkspaceDir); err != nil {
			return errors.New(fmt.Sprintf("Workspace dir '%s' doesn't exist", absWorkspaceDir))
		}

		for _, installableObj := range installables {
			// load the config file (if we've cached it)
			err := installableObj.LoadConfigFile(absWorkspaceDir)
			if err != nil {
				return errors.WithStack(err)
			}
		}
	}

	return nil
}
