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

package installable

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	"github.com/sugarkube/sugarkube/internal/pkg/acquirer"
	"github.com/sugarkube/sugarkube/internal/pkg/config"
	"github.com/sugarkube/sugarkube/internal/pkg/constants"
	"github.com/sugarkube/sugarkube/internal/pkg/convert"
	"github.com/sugarkube/sugarkube/internal/pkg/interfaces"
	"github.com/sugarkube/sugarkube/internal/pkg/log"
	"github.com/sugarkube/sugarkube/internal/pkg/printer"
	"github.com/sugarkube/sugarkube/internal/pkg/structs"
	"github.com/sugarkube/sugarkube/internal/pkg/templater"
	"github.com/sugarkube/sugarkube/internal/pkg/utils"
	"github.com/sugarkube/sugarkube/internal/pkg/vars"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

type Kapp struct {
	manifestId       string
	configFileDir    string                           // path to the directory containing the kapp's sugarkube.yaml file
	mergedDescriptor structs.KappDescriptorWithMaps   // the final descriptor after merging all the descriptor layers. This is a template until its rendered by TemplateDescriptor
	descriptorLayers []structs.KappDescriptorWithMaps // config templates where values from later configs will take precedence over earlier ones
	kappCacheDir     string                           // the top-level directory for this kapp in the workspace, i.e. the directory containing the kapp's .sugarkube directory
	localRegistry    interfaces.IRegistry             // a registry local to the kapp that contains the results of merging
	// each of its parents' registries, tailored depending on whether parent was in the same manifest
}

// Returns the non-fully qualified ID
func (k Kapp) Id() string {
	return k.mergedDescriptor.Id
}

// Returns the manifest ID
func (k Kapp) ManifestId() string {
	return k.manifestId
}

// Returns true if the installable has any pre-/post- actions
func (k Kapp) HasActions() bool {
	return len(k.PreInstallActions()) > 0 ||
		len(k.PostInstallActions()) > 0 ||
		len(k.PreDeleteActions()) > 0 ||
		len(k.PostDeleteActions()) > 0
}

// Returns the pre/post install/delete actions
func (k Kapp) getActions(pre bool, install bool) []structs.Action {
	// convert the map to a list
	postActions := make([]structs.Action, 0)

	var actions []map[string]structs.Action

	if pre {
		if install {
			actions = k.mergedDescriptor.PreInstallActions
		} else {
			actions = k.mergedDescriptor.PreDeleteActions
		}
	} else {
		if install {
			actions = k.mergedDescriptor.PostInstallActions
		} else {
			actions = k.mergedDescriptor.PostDeleteActions
		}
	}

	for _, actionMap := range actions {
		for k, v := range actionMap {
			v.Id = k
			postActions = append(postActions, v)
		}
	}

	return postActions
}

// Returns the post-install actions
func (k Kapp) PreInstallActions() []structs.Action {
	return k.getActions(true, true)
}

// Returns the post-delete actions
func (k Kapp) PreDeleteActions() []structs.Action {
	return k.getActions(true, false)
}

// Returns the post-install actions
func (k Kapp) PostInstallActions() []structs.Action {
	return k.getActions(false, true)
}

// Returns the post-delete actions
func (k Kapp) PostDeleteActions() []structs.Action {
	return k.getActions(false, false)
}

// Every time we add a new descriptor remerge the descriptor.
// If `prepend` is true the new layer will be prepended to the list of layers, otherwise it'll be appended.
// Descriptors later in the layers array will override earlier values
func (k *Kapp) AddDescriptor(config structs.KappDescriptorWithMaps, prepend bool) error {
	configLayers := k.descriptorLayers

	if configLayers == nil {
		configLayers = []structs.KappDescriptorWithMaps{}
	}

	log.Logger.Tracef("Adding new descriptor for kapp '%s' (prepend=%v): %+v", k.FullyQualifiedId(),
		prepend, config)

	// deep copy the input because otherwise Mergo mutates the global state while merging somehow (something
	// to do with maps being pointers I suspect)
	configCopy := structs.KappDescriptorWithMaps{}
	err := utils.DeepCopy(config, &configCopy)
	if err != nil {
		return errors.WithStack(err)
	}

	// set placeholder values for rendered paths of any templates
	for k, _ := range config.Templates {
		template := config.Templates[k]
		template.RenderedPath = constants.KappGeneratedPlaceholder
		config.Templates[k] = template
	}

	if prepend {
		k.descriptorLayers = append([]structs.KappDescriptorWithMaps{configCopy}, configLayers...)
	} else {
		k.descriptorLayers = append(configLayers, configCopy)
	}

	// until https://github.com/imdario/mergo/issues/90 is resolved we need to manually propagate
	// non-empty fields for maps to later layers
	// todo -  remove this once https://github.com/imdario/mergo/issues/90 is merged
	if len(k.descriptorLayers) > 1 {
		for i := 0; i < len(k.descriptorLayers)-1; i++ {
			previousLayer := k.descriptorLayers[i]
			currentLayer := k.descriptorLayers[i+1]

			if currentLayer.Sources == nil {
				currentLayer.Sources = map[string]structs.Source{}
			}

			for key, previousSource := range previousLayer.Sources {
				currentSource, ok := currentLayer.Sources[key]
				if !ok {
					// if no source exists, initialise one so we can propagate values
					currentSource = structs.Source{
						Options: map[string]interface{}{},
					}

					currentLayer.Sources = make(map[string]structs.Source)
				}

				if currentSource.Uri == "" && previousSource.Uri != "" {
					currentSource.Uri = previousSource.Uri
				}
				if currentSource.Id == "" && previousSource.Id != "" {
					currentSource.Id = previousSource.Id
				}
				for k, v := range previousSource.Options {
					if _, ok := currentSource.Options[k]; !ok {
						currentSource.Options[k] = v
					}
				}

				currentLayer.Sources[key] = currentSource
			}

			if currentLayer.Outputs == nil {
				currentLayer.Outputs = map[string]structs.Output{}
			}

			for key, previousOutput := range previousLayer.Outputs {
				currentOutput, ok := currentLayer.Outputs[key]
				if !ok {
					currentOutput = structs.Output{
						Conditions: make([]string, 0),
					}
				}

				if currentOutput.Id == "" && previousOutput.Id != "" {
					currentOutput.Id = previousOutput.Id
				}
				if currentOutput.Path == "" && previousOutput.Path != "" {
					currentOutput.Path = previousOutput.Path
				}
				if currentOutput.Format == "" && previousOutput.Format != "" {
					currentOutput.Format = previousOutput.Format
				}
				if len(currentOutput.Conditions) == 0 && len(previousOutput.Conditions) > 0 {
					currentOutput.Conditions = previousOutput.Conditions
				}

				currentLayer.Outputs[key] = currentOutput
			}

			if currentLayer.Vars == nil {
				currentLayer.Vars = map[string]interface{}{}
			}

			for k, v := range previousLayer.Vars {
				if _, ok := currentLayer.Vars[k]; !ok {
					currentLayer.Vars[k] = v
				}
			}

			if currentLayer.RunUnits == nil {
				currentLayer.RunUnits = map[string]structs.RunUnit{}
			}

			log.Logger.Debug("Manually merging run units")
			for key, previousRunUnit := range previousLayer.RunUnits {
				currentRunUnit, ok := currentLayer.RunUnits[key]
				if !ok {
					currentRunUnit = structs.RunUnit{
						EnvVars: map[string]string{},
					}
				}

				log.Logger.Tracef("Manually merging previous run unit: %#v with current run unit %#v",
					previousRunUnit, currentRunUnit)

				// hacks upon hacks. We need to initialise a map in the previous
				// layer to stop mergo causing a panic (https://github.com/imdario/mergo/issues/90)
				if previousLayer.RunUnits[key].EnvVars == nil {
					previousRunUnit.EnvVars = map[string]string{}
					previousLayer.RunUnits[key] = previousRunUnit
				}

				if currentRunUnit.WorkingDir == "" && previousRunUnit.WorkingDir != "" {
					currentRunUnit.WorkingDir = previousRunUnit.WorkingDir
				}

				if len(currentRunUnit.Conditions) == 0 && len(previousRunUnit.Conditions) > 0 {
					currentRunUnit.Conditions = previousRunUnit.Conditions
				}
				if len(currentRunUnit.Binaries) == 0 && len(previousRunUnit.Binaries) > 0 {
					currentRunUnit.Binaries = previousRunUnit.Binaries
				}

				if currentRunUnit.EnvVars == nil {
					currentRunUnit.EnvVars = map[string]string{}
				}

				for k, v := range previousRunUnit.EnvVars {
					if _, ok := currentRunUnit.EnvVars[k]; !ok {
						currentRunUnit.EnvVars[k] = v
					}
				}

				// use the current run steps if any are defined, otherwise use the previous ones
				currentRunUnit.PlanInstall = mergeRunSteps(previousRunUnit.PlanInstall, currentRunUnit.PlanInstall)
				currentRunUnit.ApplyInstall = mergeRunSteps(previousRunUnit.ApplyInstall, currentRunUnit.ApplyInstall)
				currentRunUnit.PlanDelete = mergeRunSteps(previousRunUnit.PlanDelete, currentRunUnit.PlanDelete)
				currentRunUnit.ApplyDelete = mergeRunSteps(previousRunUnit.ApplyDelete, currentRunUnit.ApplyDelete)
				currentRunUnit.Output = mergeRunSteps(previousRunUnit.Output, currentRunUnit.Output)
				currentRunUnit.Clean = mergeRunSteps(previousRunUnit.Clean, currentRunUnit.Clean)

				log.Logger.Tracef("Want to set run unit for '%s' to %#v", key, currentRunUnit)
				currentLayer.RunUnits[key] = currentRunUnit
			}

			k.descriptorLayers[i+1] = currentLayer
		}
	}

	return k.mergeDescriptorLayers()
}

// manually merges run steps (this is required due to bugs in mergo)
func mergeRunSteps(previous []structs.RunStep, current []structs.RunStep) []structs.RunStep {
	if len(current) > 0 {
		return current
	} else {
		return previous
	}
}

// Merges the descriptor layers to create a new templatable merged descriptor
func (k *Kapp) mergeDescriptorLayers() error {
	mergedDescriptor := structs.KappDescriptorWithMaps{}

	for _, layer := range k.descriptorLayers {
		log.Logger.Tracef("Merging config layer for kapp '%s' - layer %#v into existing map %#v",
			k.FullyQualifiedId(), layer, mergedDescriptor)
		err := vars.Merge(&mergedDescriptor, layer)
		if err != nil {
			return errors.WithStack(err)
		}
	}

	k.mergedDescriptor = mergedDescriptor
	log.Logger.Debugf("Set new merged descriptor for kapp '%s' to '%+v'", k.FullyQualifiedId(), mergedDescriptor)

	return nil
}

// Returns the merged descriptor, which is the result of merging all descriptors in the
// list of descriptors
func (k Kapp) GetDescriptor() structs.KappDescriptorWithMaps {
	return k.mergedDescriptor
}

// Returns the fully-qualified ID of a kapp
func (k Kapp) FullyQualifiedId() string {
	if k.manifestId == "" {
		return k.Id()
	} else {
		return strings.Join([]string{k.manifestId, k.Id()}, constants.NamespaceSeparator)
	}
}

// Returns the directory for this kapp in the cache, i.e. the directory containing the kapp's
// .sugarkube directory. This path may or may not exist depending on whether the cache has actually
// been created.
func (k Kapp) GetCacheDir() string {
	return k.kappCacheDir
}

// Returs the directory containing the kapp's sugarkube.yaml file
func (k Kapp) GetConfigFileDir() string {
	return k.configFileDir
}

// Returns an array of acquirers configured for the sources for this kapp. We need to recompute these each time
// instead of caching them so that any manifest overrides will take effect.
func (k Kapp) Acquirers() (map[string]acquirer.Acquirer, error) {

	acquirers, err := acquirer.GetAcquirersFromSources(k.mergedDescriptor.Sources)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return acquirers, nil
}

// Sets the workspace directory, i.e. the one users specify on the command line
func (k *Kapp) SetWorkspaceDir(workspaceDir string) error {
	// set the workspace dir as an absolute path
	absWorkspaceDir, err := filepath.Abs(workspaceDir)
	if err != nil {
		return errors.WithStack(err)
	}
	k.kappCacheDir = filepath.Join(absWorkspaceDir, k.manifestId, k.Id())

	return nil
}

// Loads the kapp's sugarkube.yaml file and adds it as a config layer
// workspaceDir - The path to the workspace directory. Can be an empty string if the kapp isn't in a workspace
func (k *Kapp) LoadConfigFile(workspaceDir string) error {

	err := k.SetWorkspaceDir(workspaceDir)
	if err != nil {
		return errors.WithStack(err)
	}

	configFilePaths, err := utils.FindFilesByPattern(k.GetCacheDir(), constants.KappConfigFileName,
		true, false)
	if err != nil {
		return errors.Wrapf(err, "Error finding '%s' in '%s'",
			constants.KappConfigFileName, k.GetCacheDir())
	}

	if len(configFilePaths) == 0 {
		return errors.New(fmt.Sprintf("No '%s' file found for kapp "+
			"'%s' in %s", constants.KappConfigFileName, k.FullyQualifiedId(), k.GetCacheDir()))
	} else if len(configFilePaths) > 1 {
		// todo - have a way of declaring the 'right' one in the manifest
		panic(fmt.Sprintf("Multiple '%s' found for kapp '%s'. Disambiguation "+
			"not implemented yet: %s", constants.KappConfigFileName, k.FullyQualifiedId(),
			strings.Join(configFilePaths, ", ")))
	}

	configFilePath := configFilePaths[0]
	k.configFileDir = filepath.Dir(configFilePath)

	descriptorWithLists := structs.KappDescriptorWithLists{}

	err = utils.LoadYamlFile(configFilePath, &descriptorWithLists)
	if err != nil {
		return errors.WithStack(err)
	}

	descriptorWithMaps, err := convert.KappDescriptorWithListsToMap(descriptorWithLists)
	if err != nil {
		return errors.WithStack(err)
	}

	log.Logger.Tracef("Adding descriptor to kapp '%s' for its %s file", k.FullyQualifiedId(),
		constants.KappConfigFileName)

	err = k.AddDescriptor(descriptorWithMaps, true)
	if err != nil {
		return errors.WithStack(err)
	}

	// now we've loaded the kapp's sugarkube.yaml file we can prepend descriptors for each
	// requirement declared by it that has an entry in the global config file, provided the
	// kapp hasn't opted out of them.
	if !k.GetDescriptor().IgnoreGlobalDefaults {
		// add descriptors for globally defined defaults
		descriptor := structs.KappDescriptorWithMaps{
			KappConfig: structs.KappConfig{
				RunUnits: map[string]structs.RunUnit{
					constants.ConfigFileRunUnits: config.CurrentConfig.RunUnits,
				},
			},
		}

		log.Logger.Tracef("Adding default run units to kapp '%s': %#v", k.FullyQualifiedId(), descriptor)
		err = k.AddDescriptor(descriptor, true)
		if err != nil {
			return errors.WithStack(err)
		}

		for i := len(k.GetDescriptor().Requires) - 1; i >= 0; i-- {
			requirement := k.GetDescriptor().Requires[i]
			programDescriptor, ok := config.CurrentConfig.Programs[requirement]
			if !ok {
				continue
			}

			descriptor := structs.KappDescriptorWithMaps{
				KappConfig: programDescriptor,
			}

			log.Logger.Tracef("Adding descriptor to kapp '%s' for requirement '%s'",
				k.FullyQualifiedId(), requirement)

			err = k.AddDescriptor(descriptor, true)
			if err != nil {
				return errors.WithStack(err)
			}
		}
	}

	return nil
}

// Returns a boolean indicating whether the kapp declares any outputs
func (k Kapp) HasOutputs() bool {
	return len(k.mergedDescriptor.Outputs) > 0
}

// Returns the kapps local registry, which is the result of merging all its parents' local registries,
// cleaned up to account for parents possibly being in different manifests. It doesn't include the
// global registry though.
func (k Kapp) GetLocalRegistry() interfaces.IRegistry {
	return k.localRegistry
}

// Sets the local registry for the kapp
func (k *Kapp) SetLocalRegistry(registry interfaces.IRegistry) {
	log.Logger.Tracef("Setting local registry for kapp '%s' to: %#v", k.FullyQualifiedId(), registry)
	k.localRegistry = registry
}

// Templates the kapp's merged descriptor
func (k *Kapp) TemplateDescriptor(templateVars map[string]interface{}) error {

	// remerge the layers so we've got a fresh template to render
	err := k.mergeDescriptorLayers()
	if err != nil {
		return errors.WithStack(err)
	}

	configTemplate, err := yaml.Marshal(k.mergedDescriptor)
	if err != nil {
		return errors.WithStack(err)
	}

	templateString := string(configTemplate[:])

	// todo - we should probably template the vars first, then template the descriptor in case
	//  variables are referenced from the rest of the descriptor (there was an error when
	//  a variable contained the value of the rendered path of a template, and the descriptor
	//  referenced that variable. The `templated_path` var's value was being set, but the descriptor's
	//  use of it was blank. Even using the iterative templater didn't work...

	var outBuf bytes.Buffer
	err = templater.TemplateString(templateString, &outBuf, templateVars)
	if err != nil {
		return errors.WithStack(err)
	}

	log.Logger.Tracef("Rendered merged kapp descriptor\n%#v\nto:\n%s",
		templateString, outBuf.String())

	configObj := structs.KappDescriptorWithMaps{}
	err = yaml.Unmarshal(outBuf.Bytes(), &configObj)
	if err != nil {
		return errors.Wrapf(err, "Error unmarshalling rendered merged kapp descriptor: %s",
			outBuf.String())
	}

	k.mergedDescriptor = configObj
	return nil
}

// Returns a map of all variables for the kapp
func (k Kapp) Vars(stack interfaces.IStack) (map[string]interface{}, error) {
	kappVars, err := k.getVarsFromFiles(stack.GetConfig())
	if err != nil {
		return nil, errors.WithStack(err)
	}

	kappIntrinsicDataConverted := map[string]interface{}{}

	kappIntrinsicData := k.getIntrinsicData()
	kappIntrinsicDataConverted = convert.MapStringStringToMapStringInterface(kappIntrinsicData)

	// merge kapp.Vars with the vars from files so kapp.Vars take precedence. Todo - document the order of precedence
	err = vars.Merge(&kappVars, k.mergedDescriptor.Vars)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	// namespace kapp variables. This is safer than letting kapp variables overwrite arbitrary values (e.g.
	// so they can't change the target stack, whether the kapp's approved, etc.), but may be too restrictive
	// in certain situations. We'll have to see
	kappIntrinsicDataConverted[constants.KappVarsVarsKey] = kappVars

	// convert the map of structs to a plain map. This is necessary due to an apparent bug in text/template that was
	// causing all struct fields to have '<no value>'. It also has the benefit of meaning we can key by lower snake_cased
	// field names which is less confusing.
	templatesMap := make(map[string]interface{}, 0)
	for k, template := range k.mergedDescriptor.Templates {
		templateMap := map[string]interface{}{
			"source":        template.Source,
			"dest":          template.Dest,
			"rendered_path": template.RenderedPath,
			"sensitive":     template.Sensitive,
		}
		templatesMap[k] = templateMap
	}

	kappIntrinsicDataConverted[constants.KappVarsTemplatesKey] = templatesMap

	namespacedKappMap := map[string]interface{}{
		constants.KappVarsKappKey: kappIntrinsicDataConverted,
	}

	if k.localRegistry != nil {
		// merge the local registry with the template vars so outputs are available to templates
		log.Logger.Tracef("Merging local registry for kapp '%s' with kapp vars. Local registry is: %#v",
			k.FullyQualifiedId(), k.localRegistry)

		err = vars.Merge(&namespacedKappMap, k.localRegistry.AsMap())
		if err != nil {
			return nil, errors.WithStack(err)
		}
	}

	log.Logger.Tracef("Returning vars for kapp '%s': %v", k.FullyQualifiedId(), namespacedKappMap)

	return namespacedKappMap, nil
}

// Returns certain kapp data that should be exposed as variables when running kapps
func (k Kapp) getIntrinsicData() map[string]string {
	return map[string]string{
		"id":         k.Id(),
		"manifestId": k.ManifestId(),
		"cacheRoot":  k.GetCacheDir(),
	}
}

// Finds all vars files for the given kapp and returns the result of merging
// all the data.
func (k Kapp) getVarsFromFiles(stackConfig interfaces.IStackConfig) (map[string]interface{}, error) {
	dirs, err := k.findVarsFiles(stackConfig)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	values := map[string]interface{}{}

	err = vars.MergePaths(&values, dirs...)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return values, nil
}

// This searches a directory tree from a given root path for files whose values
// should be merged together for a kapp. If a kapp instance is supplied, additional files
// will be searched for, in addition to stack-specific ones.
func (k Kapp) findVarsFiles(stackConfig interfaces.IStackConfig) ([]string, error) {
	precedence := []string{
		utils.StripExtension(constants.ValuesFile),
		stackConfig.GetName(),
		stackConfig.GetProvider(),
		stackConfig.GetProvisioner(),
		stackConfig.GetAccount(),
		stackConfig.GetRegion(),
		stackConfig.GetProfile(),
		stackConfig.GetCluster(),
		constants.ProfileDir,
		constants.ClusterDir,
	}

	var kappId string

	// prepend the kapp ID to the precedence array
	precedence = append([]string{k.Id()}, precedence...)

	acquirers, err := k.Acquirers()
	if err != nil {
		return nil, errors.WithStack(err)
	}

	for _, acquirerObj := range acquirers {
		precedence = append(precedence, acquirerObj.Id())

		id, err := acquirerObj.FullyQualifiedId()
		if err != nil {
			return nil, errors.WithStack(err)
		}

		precedence = append(precedence, id)
	}

	kappId = k.Id()

	paths := make([]string, 0)

	for _, searchDir := range stackConfig.KappVarsDirs() {
		searchPath, err := filepath.Abs(filepath.Join(stackConfig.GetDir(), searchDir))
		if err != nil {
			return nil, errors.WithStack(err)
		}

		log.Logger.Infof("Searching for files/dirs for kapp '%s' under "+
			"'%s' with basenames: %s", kappId, searchPath,
			strings.Join(precedence, ", "))

		err = utils.PrecedenceWalk(searchPath, precedence, func(path string,
			info os.FileInfo, err error) error {
			if err != nil {
				return errors.WithStack(err)
			}

			if !info.IsDir() {
				ext := filepath.Ext(path)

				if strings.ToLower(ext) != ".yaml" {
					log.Logger.Debugf("Ignoring non-yaml file: %s", path)
					return nil
				}

				log.Logger.Debugf("Adding kapp var file: %s", path)
				paths = append(paths, path)
			}

			return nil
		})

		if err != nil {
			return nil, errors.WithStack(err)
		}
	}

	log.Logger.Debugf("Kapp var paths for kapp '%s' are: %s", kappId,
		strings.Join(paths, ", "))

	return paths, nil
}

// Renders templates for the kapp and returns the paths they were written to
func (k *Kapp) RenderTemplates(templateVars map[string]interface{}, stackConfig interfaces.IStackConfig,
	dryRun bool) error {

	dryRunPrefix := ""
	if dryRun {
		dryRunPrefix = "[Dry run] "
	}

	// make sure the cache dir exists
	if _, err := os.Stat(k.GetCacheDir()); err != nil {
		return errors.New(fmt.Sprintf("Cache dir '%s' doesn't exist",
			k.GetCacheDir()))
	}

	if len(k.mergedDescriptor.Templates) == 0 {
		log.Logger.Infof("%sNo templates to render for kapp '%s'", dryRunPrefix, k.FullyQualifiedId())
		return nil
	}

	log.Logger.Infof("%sRendering templates for kapp '%s'", dryRunPrefix, k.FullyQualifiedId())

	// build a list of rendered templates so we can add a new config descriptor that will contain the rendered paths
	renderedTemplates := make(map[string]structs.Template, 0)

	for templateId, templateDefinition := range k.mergedDescriptor.Templates {
		// make sure that if any conditions are declared that they're all true
		if len(templateDefinition.Conditions) > 0 {
			allOk, err := utils.All(templateDefinition.Conditions)
			if err != nil {
				return errors.WithStack(err)
			}

			if !allOk {
				log.Logger.Infof("Skipping rendering template '%s' for kapp '%s' because some conditions "+
					"evaluated to false", templateId, k.FullyQualifiedId())
				continue
			}
		}

		rawTemplateSource := templateDefinition.Source

		if rawTemplateSource == "" {
			return errors.New(fmt.Sprintf("Template %s has an empty source: %+v", templateId, templateDefinition))
		}

		log.Logger.Debugf("Template '%s' has a definition: %+v", templateId, templateDefinition)

		// run the source path through the templater in case it contains variables
		templateSource, err := templater.RenderTemplate(rawTemplateSource, templateVars)
		if err != nil {
			return errors.WithStack(err)
		}

		if !filepath.IsAbs(templateSource) {
			foundTemplate := false

			// see whether the template is in the kapp itself
			possibleSource := filepath.Join(k.configFileDir, templateSource)
			log.Logger.Debugf("Searching for kapp template '%s' in '%s'", templateSource, possibleSource)
			_, err := os.Stat(possibleSource)
			if err == nil {
				templateSource = possibleSource
				foundTemplate = true
			}

			if !foundTemplate {
				// search each template directory defined in the stack config
				for _, templateDir := range stackConfig.TemplateDirs() {
					possibleSource := filepath.Join(stackConfig.GetDir(), templateDir, templateSource)
					log.Logger.Debugf("Searching for kapp template in '%s'", possibleSource)
					_, err := os.Stat(possibleSource)
					if err == nil {
						templateSource = possibleSource
						foundTemplate = true
						break
					}
				}
			}

			if foundTemplate {
				log.Logger.Debugf("Found template at %s", templateSource)
			} else {
				return errors.New(fmt.Sprintf("Failed to find template '%s' "+
					"in any of the defined template directories: %s", templateSource,
					strings.Join(stackConfig.TemplateDirs(), ", ")))
			}
		}

		if !filepath.IsAbs(templateSource) {
			templateSource, err = filepath.Abs(templateSource)
			if err != nil {
				return errors.WithStack(err)
			}
		}

		log.Logger.Debugf("%sTemplating file '%s' with vars: %#v", dryRunPrefix,
			templateSource, templateVars)

		rawDestPath := templateDefinition.Dest
		// run the dest path through the templater in case it contains variables
		destPath, err := templater.RenderTemplate(rawDestPath, templateVars)
		if err != nil {
			return errors.WithStack(err)
		}

		if !filepath.IsAbs(destPath) {
			destPath = filepath.Join(k.configFileDir, destPath)
		}

		// check whether the dest path exists
		if _, err := os.Stat(destPath); err == nil {
			log.Logger.Infof("%sTemplate destination path '%s' exists. "+
				"File will be overwritten by rendered template '%s' for kapp '%s'", dryRunPrefix,
				destPath, templateSource, k.Id())
		}

		// check whether the parent directory for dest path exists and return an error if not
		destDir := filepath.Dir(destPath)
		if _, err := os.Stat(destDir); os.IsNotExist(err) {
			log.Logger.Infof("Destination template directory '%s' doesn't exist", destDir)
			return errors.New(fmt.Sprintf("Can't write template '%s' to non-existent directory: %s", templateId, destDir))
		}

		var outBuf bytes.Buffer

		err = templater.TemplateFile(templateSource, &outBuf, templateVars)
		if err != nil {
			return errors.WithStack(err)
		}

		log.Logger.Infof("%sWriting rendered template '%s' for kapp "+
			"'%s' to '%s'", dryRunPrefix, templateSource, k.FullyQualifiedId(), destPath)
		log.Logger.Tracef("%sTemplate rendered as:\n%s", dryRunPrefix, outBuf.String())

		if !dryRun {
			err := ioutil.WriteFile(destPath, outBuf.Bytes(), 0644)
			if err != nil {
				return errors.WithStack(err)
			}
		}

		template := structs.Template{
			Source:       templateDefinition.Source,
			Dest:         templateDefinition.Dest,
			RenderedPath: destPath,
			Sensitive:    templateDefinition.Sensitive,
		}

		renderedTemplates[templateId] = template
	}

	descriptor := structs.KappDescriptorWithMaps{
		KappConfig: structs.KappConfig{
			Templates: renderedTemplates,
		},
	}

	// add a new descriptor containing the rendered template paths
	err := k.AddDescriptor(descriptor, false)
	if err != nil {
		return errors.WithStack(err)
	}

	return nil
}

// Loads outputs for the kapp, parses and returns them
func (k Kapp) GetOutputs(ignoreMissing bool, dryRun bool) (map[string]interface{}, error) {
	outputs := map[string]interface{}{}

	dryRunPrefix := ""
	if dryRun {
		dryRunPrefix = "[Dry run] "
	}

	var err error

	var allOk bool

	for _, output := range k.mergedDescriptor.Outputs {
		allOk, err = utils.All(output.Conditions)
		if err != nil {
			return nil, errors.WithStack(err)
		}

		if !allOk {
			log.Logger.Infof("Not loading output '%s' for kapp '%s' which has failed conditions",
				output.Id, k.FullyQualifiedId())

			_, err := printer.Fprintf("[yellow]Not loading output '%s' for kapp "+
				"'[bold]%s[reset][yellow]' due to failed conditions\n", output.Id, k.FullyQualifiedId())
			if err != nil {
				return nil, errors.WithStack(err)
			}

			continue
		}

		// if the output exists, parse it as the declared type and put it in the map
		path := output.Path

		// prepend the config directory if a relative path was given
		if !strings.HasPrefix(path, "/") {
			path, err = filepath.Abs(filepath.Join(k.configFileDir, output.Path))
			if err != nil {
				return nil, errors.WithStack(err)
			}
		}

		if !dryRun {
			if _, err = os.Stat(path); err != nil {
				if ignoreMissing {
					_, err := printer.Fprintf("[yellow]Ignoring missing output '%s' for kapp "+
						"'[bold]%s[reset][yellow]'\n", path, k.FullyQualifiedId())
					if err != nil {
						return nil, errors.WithStack(err)
					}
					outputs[output.Id] = nil
					continue
				} else {
					return nil, errors.WithStack(err)
				}
			}
		}

		log.Logger.Infof("%sLoading output '%s' from kapp '%s' at '%s' as %s", dryRunPrefix,
			output.Id, k.FullyQualifiedId(), path, output.Format)

		var parsedOutput interface{}

		switch strings.ToLower(output.Format) {
		case "json":
			if !dryRun {
				rawJson, err := ioutil.ReadFile(path)
				if err != nil {
					return nil, errors.WithStack(err)
				}

				err = json.Unmarshal(rawJson, &parsedOutput)
				if err != nil {
					return nil, errors.WithStack(err)
				}
			}
		case "yaml":
			if !dryRun {
				err = utils.LoadYamlFile(path, &parsedOutput)
				if err != nil {
					return nil, errors.WithStack(err)
				}

				parsedOutput, err = convert.MapInterfaceInterfaceToMapStringInterface(parsedOutput.(map[interface{}]interface{}))
				if err != nil {
					return nil, errors.WithStack(err)
				}
			}
		case "text":
			if !dryRun {
				byteOutput, err := ioutil.ReadFile(path)
				if err != nil {
					return nil, errors.WithStack(err)
				}
				parsedOutput = string(byteOutput)
			}
		default:
			return nil, errors.New(fmt.Sprintf("Unsupported output format '%s' for kapp '%s'",
				output.Format, k.FullyQualifiedId()))
		}

		outputs[output.Id] = parsedOutput

		// if it's sensitive, delete it
		if output.Sensitive {
			log.Logger.Infof("%sDeleting sensitive output file: %s", dryRunPrefix, path)
			if !dryRun {
				err = os.Remove(path)
				if err != nil {
					return nil, errors.WithStack(err)
				}
			}
		}
	}

	return outputs, nil
}
