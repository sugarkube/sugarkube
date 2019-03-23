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
	"github.com/pkg/errors"
	"github.com/sugarkube/sugarkube/internal/pkg/clustersot"
	"github.com/sugarkube/sugarkube/internal/pkg/config"
	"github.com/sugarkube/sugarkube/internal/pkg/convert"
	"github.com/sugarkube/sugarkube/internal/pkg/installable"
	"github.com/sugarkube/sugarkube/internal/pkg/interfaces"
	"github.com/sugarkube/sugarkube/internal/pkg/log"
	"github.com/sugarkube/sugarkube/internal/pkg/provider"
	"github.com/sugarkube/sugarkube/internal/pkg/provisioner"
	"github.com/sugarkube/sugarkube/internal/pkg/registry"
	"github.com/sugarkube/sugarkube/internal/pkg/templater"
	"github.com/sugarkube/sugarkube/internal/pkg/vars"
	"gopkg.in/yaml.v2"
)

// Top-level struct that holds references to instantiations of other objects
// we need to pass around. This is in its own package to avoid circular
// dependencies.
type Stack struct {
	GlobalConfig *config.Config // config loaded for the program from the 'sugarkube-conf.yaml' file
	Config       *StackConfig
	Provider     provider.Provider
	Provisioner  provisioner.Provisioner
	Status       *ClusterStatus
	registry     *registry.Registry
}

// Creates a new Stack
func newStack(globalConfig *config.Config, config *StackConfig,
	provider provider.Provider, registry *registry.Registry) (*Stack, error) {

	stack := &Stack{
		GlobalConfig: globalConfig,
		Config:       config,
		Provider:     provider,
		Provisioner:  nil,
		Status: &ClusterStatus{
			isOnline:              false,
			isReady:               false,
			sleepBeforeReadyCheck: 0,
			startedThisRun:        false,
		},
		registry: registry,
	}

	clusterSot, err := clustersot.NewClusterSot(clustersot.KUBECTL, stack)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	provisionerImpl, err := provisioner.New(stack.Config.Provisioner(), stack, clusterSot)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	stack.Provisioner = provisionerImpl

	return stack, nil
}

func (s Stack) GetConfig() interfaces.IStackConfig {
	return s.Config
}

func (s Stack) GetGlobalConfig() *config.Config {
	return s.GlobalConfig
}

func (s Stack) GetStatus() interfaces.IClusterStatus {
	return s.Status
}

func (s Stack) GetRegistry() *registry.Registry {
	return s.registry
}

// Merges and templates vars from all configured sources. If an installable instance
// is given, data specific to it will be included in the returned map,
// otherwise only stack-specific variables will be returned.
func (s *Stack) TemplatedVars(installableObj installable.Installable,
	installerVars map[string]interface{}) (map[string]interface{}, error) {

	stackConfig := s.Config

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
		stackConfig.Provider(),
		stackConfig.Account(), // may be blank depending on the provider
		stackConfig.Profile(),
		stackConfig.Cluster(),
		stackConfig.Region(), // may be blank depending on the provider
	}

	configFragments = append(configFragments, map[string]interface{}{
		"sugarkube": installerVars,
	})

	configFragments = append(configFragments, stackConfig.GetProviderVars())

	// merge in values from the registry
	log.Logger.Tracef("Merging stack vars with registry: %v", s.registry)
	configFragments = append(configFragments, s.registry.AsMap())

	if installableObj != nil {
		installableVars, err := installableObj.Vars()
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
