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
	kappCacheDir     string                           // the top-level directory for this kapp in the cache, i.e. the directory containing the kapp's .sugarkube directory
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

func (k Kapp) State() string {
	return k.mergedDescriptor.State
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

	if prepend {
		k.descriptorLayers = append([]structs.KappDescriptorWithMaps{configCopy}, configLayers...)
	} else {
		// until https://github.com/imdario/mergo/issues/90 is resolved we need to manually propagate
		// non-empty fields for maps to later layers
		// todo -  remove this once https://github.com/imdario/mergo/issues/90 is merged
		if len(k.descriptorLayers) > 0 {
			previousLayer := k.descriptorLayers[len(k.descriptorLayers)-1]

			for key, previousSource := range previousLayer.Sources {
				currentSource, ok := configCopy.Sources[key]
				if !ok {
					continue
				}

				if currentSource.Uri == "" && previousSource.Uri != "" {
					currentSource.Uri = previousSource.Uri
				}

				if currentSource.Id == "" && previousSource.Id != "" {
					currentSource.Id = previousSource.Id
				}

				configCopy.Sources[key] = currentSource
			}

			for key, previousOutput := range previousLayer.Outputs {
				currentOutput, ok := configCopy.Outputs[key]
				if !ok {
					continue
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

				configCopy.Outputs[key] = currentOutput
			}
		}

		k.descriptorLayers = append(configLayers, configCopy)
	}

	return k.mergeDescriptorLayers()
}

// Merges the descriptor layers to create a new templatable merged descriptor
func (k *Kapp) mergeDescriptorLayers() error {
	mergedDescriptor := structs.KappDescriptorWithMaps{}

	for _, layer := range k.descriptorLayers {
		log.Logger.Debugf("Merging config layer for kapp '%s' - layer %#v into existing map %#v",
			k.FullyQualifiedId(), layer, mergedDescriptor)
		err := vars.MergeWithStrategy(&mergedDescriptor, layer)
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

// Return env vars
func (k Kapp) GetEnvVars() map[string]interface{} {
	return k.mergedDescriptor.EnvVars
}

// Return CLI args for the Kapp for the given installer and command/target
func (k Kapp) GetCliArgs(installerName string, command string) []string {
	installerArgs, ok := k.mergedDescriptor.Args[installerName]
	if !ok {
		return []string{}
	}

	commandArgs, ok := installerArgs[command]
	if !ok {
		return []string{}
	}

	cliArgs := make([]string, 0)

	for name, value := range commandArgs {
		joined := strings.Join([]string{name, value}, "=")
		cliArgs = append(cliArgs, joined)
	}

	return cliArgs
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

// Sets the top-level cache directory, i.e. the one users specify on the command line
func (k *Kapp) SetTopLevelCacheDir(cacheDir string) error {
	// set the top level cache dir as an absolute path
	absCacheDir, err := filepath.Abs(cacheDir)
	if err != nil {
		return errors.WithStack(err)
	}
	k.kappCacheDir = filepath.Join(absCacheDir, k.manifestId, k.Id())

	return nil
}

// Loads the kapp's sugarkube.yaml file and adds it as a config layer
// cacheDir - The path to the top-level cache directory. Can be an empty string if the kapp isn't cached
func (k *Kapp) LoadConfigFile(cacheDir string) error {

	err := k.SetTopLevelCacheDir(cacheDir)
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
	// requirement declared by it that has an entry in the global config file
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
	err = vars.MergeWithStrategy(&kappVars, k.mergedDescriptor.Vars)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	// namespace kapp variables. This is safer than letting kapp variables overwrite arbitrary values (e.g.
	// so they can't change the target stack, whether the kapp's approved, etc.), but may be too restrictive
	// in certain situations. We'll have to see
	kappIntrinsicDataConverted[constants.KappVarsVarsKey] = kappVars

	// add placeholders templated paths so kapps that use them work when running
	// `kapp vars`, etc.
	templatePlaceholders := make([]string, len(k.mergedDescriptor.Templates))

	for i, _ := range k.mergedDescriptor.Templates {
		templatePlaceholders[i] = "<generated>"
	}
	kappIntrinsicDataConverted[constants.KappVarsTemplatesKey] = templatePlaceholders

	namespacedKappMap := map[string]interface{}{
		constants.KappVarsKappKey: kappIntrinsicDataConverted,
	}

	if k.localRegistry != nil {
		// merge the local registry with the template vars so outputs are available to templates
		log.Logger.Tracef("Merging local registry for kapp '%s' with kapp vars. Local registry is: %#v",
			k.FullyQualifiedId(), k.localRegistry)

		err = vars.MergeWithStrategy(&namespacedKappMap, k.localRegistry.AsMap())
		if err != nil {
			return nil, errors.WithStack(err)
		}
	}

	return namespacedKappMap, nil
}

// Returns certain kapp data that should be exposed as variables when running kapps
func (k Kapp) getIntrinsicData() map[string]string {
	return map[string]string{
		"id":        k.Id(),
		"state":     k.State(),
		"cacheRoot": k.GetCacheDir(),
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
	dryRun bool) ([]string, error) {

	dryRunPrefix := ""
	if dryRun {
		dryRunPrefix = "[Dry run] "
	}

	// make sure the cache dir exists
	if _, err := os.Stat(k.GetCacheDir()); err != nil {
		return nil, errors.New(fmt.Sprintf("Cache dir '%s' doesn't exist",
			k.GetCacheDir()))
	}

	renderedPaths := make([]string, 0)

	if len(k.mergedDescriptor.Templates) == 0 {
		log.Logger.Infof("%sNo templates to render for kapp '%s'", dryRunPrefix, k.FullyQualifiedId())
		return renderedPaths, nil
	}

	log.Logger.Infof("%sRendering templates for kapp '%s'", dryRunPrefix, k.FullyQualifiedId())

	for _, templateDefinition := range k.mergedDescriptor.Templates {
		rawTemplateSource := templateDefinition.Source

		if rawTemplateSource == "" {
			return nil, errors.New(fmt.Sprintf("Template has an empty source: %+v", templateDefinition))
		}

		log.Logger.Debugf("Template definition: %+v", templateDefinition)

		// run the source path through the templater in case it contains variables
		templateSource, err := templater.RenderTemplate(rawTemplateSource, templateVars)
		if err != nil {
			return nil, errors.WithStack(err)
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
				return renderedPaths, errors.New(fmt.Sprintf("Failed to find template '%s' "+
					"in any of the defined template directories: %s", templateSource,
					strings.Join(stackConfig.TemplateDirs(), ", ")))
			}
		}

		if !filepath.IsAbs(templateSource) {
			templateSource, err = filepath.Abs(templateSource)
			if err != nil {
				return renderedPaths, errors.WithStack(err)
			}
		}

		log.Logger.Debugf("%sTemplating file '%s' with vars: %#v", dryRunPrefix,
			templateSource, templateVars)

		rawDestPath := templateDefinition.Dest
		// run the dest path through the templater in case it contains variables
		destPath, err := templater.RenderTemplate(rawDestPath, templateVars)
		if err != nil {
			return nil, errors.WithStack(err)
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
			return renderedPaths, errors.New(fmt.Sprintf("Can't write template to non-existent directory: %s", destDir))
		}

		var outBuf bytes.Buffer

		err = templater.TemplateFile(templateSource, &outBuf, templateVars)
		if err != nil {
			return renderedPaths, errors.WithStack(err)
		}

		log.Logger.Infof("%sWriting rendered template '%s' for kapp "+
			"'%s' to '%s'", dryRunPrefix, templateSource, k.FullyQualifiedId(), destPath)
		log.Logger.Tracef("%sTemplate rendered as:\n%s", dryRunPrefix, outBuf.String())

		if !dryRun {
			err := ioutil.WriteFile(destPath, outBuf.Bytes(), 0644)
			if err != nil {
				return renderedPaths, errors.WithStack(err)
			}
		}

		renderedPaths = append(renderedPaths, destPath)
	}

	return renderedPaths, nil
}

// Loads outputs for the kapp, parses and returns them
func (k Kapp) GetOutputs(ignoreMissing bool, dryRun bool) (map[string]interface{}, error) {
	outputs := map[string]interface{}{}

	dryRunPrefix := ""
	if dryRun {
		dryRunPrefix = "[Dry run] "
	}

	for _, output := range k.mergedDescriptor.Outputs {
		// if the output exists, parse it as the declared type and put it in the map
		path, err := filepath.Abs(filepath.Join(k.configFileDir, output.Path))
		if err != nil {
			return nil, errors.WithStack(err)
		}

		if !dryRun {
			if _, err = os.Stat(path); err != nil {
				if ignoreMissing {
					log.Logger.Infof("Ignoring missing output '%s'", path)
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
			break
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
			break
		case "text":
			if !dryRun {
				byteOutput, err := ioutil.ReadFile(path)
				if err != nil {
					return nil, errors.WithStack(err)
				}
				parsedOutput = string(byteOutput)
			}
			break
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
