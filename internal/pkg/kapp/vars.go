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

package kapp

import (
	"github.com/imdario/mergo"
	"github.com/pkg/errors"
	"github.com/sugarkube/sugarkube/internal/pkg/convert"
	"github.com/sugarkube/sugarkube/internal/pkg/log"
	"gopkg.in/yaml.v2"
)

// Merges all vars for a kapp from different sources. These can be used as template
// values or as env vars/parameters to be passed to the kapp at runtime.
func MergeVarsForKapp(kappObj *Kapp, stackConfig *StackConfig,
	providerVars map[string]interface{}, sugarkubeVars map[string]interface{}) (map[string]interface{}, error) {

	rawStackConfigMap := stackConfig.AsMap()
	// convert the map to the appropriate type and namespace it
	stackConfigMap := map[string]interface{}{
		"stack": convert.MapStringStringToMapStringInterface(rawStackConfigMap),
	}

	mergedVars := map[string]interface{}{}
	err := mergo.Merge(&mergedVars, stackConfigMap, mergo.WithOverride)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	defaultVars := []string{
		stackConfig.Provider,
		stackConfig.Account, // may be blank depending on the provider
		stackConfig.Profile,
		stackConfig.Cluster,
		stackConfig.Region, // may be blank depending on the provider
	}

	// store additional runtime values under the "sugarkube" key
	sugarkubeVars["defaultVars"] = defaultVars
	mergedVars["sugarkube"] = sugarkubeVars

	err = mergo.Merge(&mergedVars, providerVars, mergo.WithOverride)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	kappMap := kappObj.AsMap()
	kappVars, err := stackConfig.GetKappVars(kappObj)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	namespacedKappMap := map[string]interface{}{
		"kapp": convert.MapStringStringToMapStringInterface(kappMap),
	}
	err = mergo.Merge(&mergedVars, namespacedKappMap, mergo.WithOverride)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	err = mergo.Merge(&mergedVars, kappVars, mergo.WithOverride)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	yamlData, err := yaml.Marshal(&mergedVars)
	log.Logger.Debugf("All merged vars:\n%s", yamlData)

	return mergedVars, nil
}
