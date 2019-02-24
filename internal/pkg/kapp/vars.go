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
	"github.com/sugarkube/sugarkube/internal/pkg/templater"
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

	kappIntrinsicData := kappObj.GetIntrinsicData()
	kappVars, err := stackConfig.GetKappVars(kappObj)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	namespacedKappMap := map[string]interface{}{
		"kapp": convert.MapStringStringToMapStringInterface(kappIntrinsicData),
	}
	err = mergo.Merge(&mergedVars, namespacedKappMap, mergo.WithOverride)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	err = mergo.Merge(&mergedVars, kappVars, mergo.WithOverride)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	templatedVars, err := templateVars(mergedVars)

	yamlData, err := yaml.Marshal(&templatedVars)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	log.Logger.Debugf("Vars after merging and templating:\n%s", yamlData)

	return templatedVars, nil
}

// Iterate over the input variables trying to replace data as if it was a template. Keep iterating up to a maximum
// number of times, or until the size of the input and output remain the same
func templateVars(vars map[string]interface{}) (map[string]interface{}, error) {

	// maximum number of iterations whils templating variables
	maxIterations := 20

	var renderedYaml string

	for i := 0; i < maxIterations; i++ {
		log.Logger.Debugf("Templating variables. Iteration %d (of max %d)", i, maxIterations)

		// convert the input variables to YAML to simplify templating it
		yamlData, err := yaml.Marshal(&vars)
		if err != nil {
			return nil, errors.WithStack(err)
		}
		log.Logger.Debugf("Vars to template (raw): %s", vars)
		log.Logger.Debugf("Vars to template as YAML:\n%s", yamlData)

		renderedYaml, err = templater.RenderTemplate(string(yamlData[:]), vars)
		log.Logger.Debugf("Variables templated as:\n%s", renderedYaml)

		// unmarshal the rendered template ready for another iteration
		var renderedVars map[string]interface{}
		err = yaml.UnmarshalStrict([]byte(renderedYaml), &renderedVars)
		if err != nil {
			return nil, errors.WithStack(err)
		}

		// todo - optimise this loop by breaking if these are equal (so no more variables were interpolated)
		vars = renderedVars
	}

	return vars, nil
}
