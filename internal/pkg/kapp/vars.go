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
	"bytes"
	"github.com/imdario/mergo"
	"github.com/pkg/errors"
	"github.com/sugarkube/sugarkube/internal/pkg/convert"
	"github.com/sugarkube/sugarkube/internal/pkg/log"
	"github.com/sugarkube/sugarkube/internal/pkg/templater"
	"gopkg.in/yaml.v2"
)

// Merges and templates vars from all configured sources. If a kapp instance
// is given, data specific to the kapp will be included in the returned map,
// otherwise only stack-specific variables will be returned.
func MergeVarsForKapp(kappObj *Kapp, stackConfig *StackConfig,
	installerVars map[string]interface{}) (map[string]interface{}, error) {

	stackIntrinsicData := stackConfig.GetIntrinsicData()
	// convert the map to the appropriate type and namespace it
	stackConfigMap := map[string]interface{}{
		"stack": convert.MapStringStringToMapStringInterface(stackIntrinsicData),
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
	installerVars["defaultVars"] = defaultVars
	mergedVars["sugarkube"] = installerVars

	err = mergo.Merge(&mergedVars, stackConfig.GetProviderVars(), mergo.WithOverride)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	kappVars, err := stackConfig.GetVarsFromFiles(kappObj)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	kappIntrinsicDataConverted := map[string]interface{}{}

	if kappObj != nil {
		kappIntrinsicData := kappObj.GetIntrinsicData()
		kappIntrinsicDataConverted = convert.MapStringStringToMapStringInterface(kappIntrinsicData)

		// merge kapp.Vars with the vars from files so kapp.Vars take precedence. Todo - document the order of precedence
		err = mergo.Merge(&kappVars, kappObj.Vars, mergo.WithOverride)
		if err != nil {
			return nil, errors.WithStack(err)
		}
	}

	// namespace kapp variables. This is safer than letting kapp variables overwrite arbitrary values (e.g.
	// so they can't change the target stack, whether the kapp's approved, etc.), but may be too restrictive
	// in certain situations. We'll have to see
	kappIntrinsicDataConverted["vars"] = kappVars

	namespacedKappMap := map[string]interface{}{
		"kapp": kappIntrinsicDataConverted,
	}
	err = mergo.Merge(&mergedVars, namespacedKappMap, mergo.WithOverride)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	templatedVars, err := templateVars(mergedVars)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	yamlData, err := yaml.Marshal(&templatedVars)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	log.Logger.Debugf("Vars after merging and templating:\n%s", yamlData)

	return templatedVars, nil
}

// Iterate over the input variables trying to replace data as if it was a template. Keep iterating up to a maximum
// number of times, or until the size of the input and output remain the same. Doing this allows us to define
// intermediate variables or aliases (e.g. set `cluster_name` = '{{ .stack.region }}-{{ .stack.account }}' then just
// use '{{ .kapp.vars.cluster_name }}'. Templating this requires 2 iterations).
func templateVars(vars map[string]interface{}) (map[string]interface{}, error) {

	// maximum number of iterations whils templating variables
	maxIterations := 20

	var previousBytes []byte
	var renderedYaml string

	for i := 0; i < maxIterations; i++ {
		log.Logger.Debugf("Templating variables. Iteration %d of max %d", i, maxIterations)

		// convert the input variables to YAML to simplify templating it
		yamlData, err := yaml.Marshal(&vars)
		if err != nil {
			return nil, errors.WithStack(err)
		}
		log.Logger.Debugf("Vars to template (raw): %s", vars)
		log.Logger.Debugf("Vars to template as YAML:\n%s", yamlData)

		renderedYaml, err = templater.RenderTemplate(string(yamlData[:]), vars)
		if err != nil {
			return nil, errors.WithStack(err)
		}
		log.Logger.Debugf("Variables templated as:\n%s", renderedYaml)

		// unmarshal the rendered template ready for another iteration
		currentBytes := []byte(renderedYaml)
		var renderedVars map[string]interface{}
		err = yaml.UnmarshalStrict(currentBytes, &renderedVars)
		if err != nil {
			return nil, errors.WithStack(err)
		}

		vars = renderedVars
		if previousBytes != nil && bytes.Equal(previousBytes, currentBytes) {
			log.Logger.Debugf("Breaking out of templating variables after %d iterations", i)
			break
		}

		previousBytes = currentBytes
	}

	return vars, nil
}
