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
	"fmt"
	"github.com/imdario/mergo"
	"github.com/pkg/errors"
	"github.com/sugarkube/sugarkube/internal/pkg/acquirer"
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
	mergedDescriptor structs.KappDescriptorWithMaps   // the final descriptor after merging all the descriptor layers. This is a template until its rendered by TemplateDescriptor
	descriptorLayers []structs.KappDescriptorWithMaps // config templates where values from later configs will take precedence over earlier ones
	kappCacheDir     string                           // the top-level directory for this kapp in the cache, i.e. the directory containing the kapp's .sugarkube directory
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

func (k Kapp) PostActions() []string {
	return k.mergedDescriptor.PostActions
}

// Every time we add a new descriptor remerge the descriptor.
// If `prepend` is true the new layer will be prepended to the list of layers, otherwise it'll be appended.
// Descriptors later in the layers array will override earlier values
func (k *Kapp) AddDescriptor(config structs.KappDescriptorWithMaps, prepend bool) error {
	configLayers := k.descriptorLayers

	if configLayers == nil {
		configLayers = []structs.KappDescriptorWithMaps{}
	}

	if prepend {
		k.descriptorLayers = append([]structs.KappDescriptorWithMaps{config}, configLayers...)
	} else {
		// until https://github.com/imdario/mergo/issues/90 is resolved we need to manually propagate
		// non-empty fields for maps to later layers
		// todo -  remove this once https://github.com/imdario/mergo/issues/90 is merged
		if len(k.descriptorLayers) > 0 {
			previousLayer := k.descriptorLayers[len(k.descriptorLayers)-1]

			for key, previousSource := range previousLayer.Sources {
				currentSource, ok := config.Sources[key]
				if !ok {
					continue
				}

				if currentSource.Uri == "" && previousSource.Uri != "" {
					currentSource.Uri = previousSource.Uri
				}

				if currentSource.Id == "" && previousSource.Id != "" {
					currentSource.Id = previousSource.Id
				}

				config.Sources[key] = currentSource
			}

			for key, previousOutput := range previousLayer.Outputs {
				currentOutput, ok := config.Outputs[key]
				if !ok {
					continue
				}

				if currentOutput.Id == "" && previousOutput.Id != "" {
					currentOutput.Id = previousOutput.Id
				}
				if currentOutput.Path == "" && previousOutput.Path != "" {
					currentOutput.Path = previousOutput.Path
				}
				if currentOutput.Type == "" && previousOutput.Type != "" {
					currentOutput.Type = previousOutput.Type
				}

				config.Outputs[key] = currentOutput
			}
		}

		k.descriptorLayers = append(configLayers, config)
	}

	return k.mergeDescriptorLayers()
}

// Merges the descriptor layers to create a new templatable merged descriptor
func (k *Kapp) mergeDescriptorLayers() error {
	mergedDescriptor := structs.KappDescriptorWithMaps{}

	for _, layer := range k.descriptorLayers {
		log.Logger.Debugf("Merging config layer %#v into existing map %#v for "+
			"kapp %s", layer, mergedDescriptor, k.FullyQualifiedId())
		err := mergo.Merge(&mergedDescriptor, layer, mergo.WithOverride)
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

	for _, arg := range commandArgs {
		joined := strings.Join([]string{arg["name"], arg["value"]}, "=")
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

	descriptorWithLists := structs.KappDescriptorWithLists{}

	err = utils.LoadYamlFile(configFilePath, &descriptorWithLists)
	if err != nil {
		return errors.WithStack(err)
	}

	descriptorWithMaps, err := convert.KappDescriptorWithListsToMap(descriptorWithLists)
	if err != nil {
		return errors.WithStack(err)
	}

	err = k.AddDescriptor(descriptorWithMaps, true)
	if err != nil {
		return errors.WithStack(err)
	}

	return nil
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

	config := structs.KappDescriptorWithMaps{}
	err = yaml.Unmarshal(outBuf.Bytes(), &config)
	if err != nil {
		return errors.Wrapf(err, "Error unmarshalling rendered merged kapp descriptor: %s",
			outBuf.String())
	}

	k.mergedDescriptor = config
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
	err = mergo.Merge(&kappVars, k.mergedDescriptor.Vars, mergo.WithOverride)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	// namespace kapp variables. This is safer than letting kapp variables overwrite arbitrary values (e.g.
	// so they can't change the target stack, whether the kapp's approved, etc.), but may be too restrictive
	// in certain situations. We'll have to see
	kappIntrinsicDataConverted["vars"] = kappVars

	// add placeholders templated paths so kapps that use them work when running
	// `kapp vars`, etc.
	templatePlaceholders := make([]string, len(k.mergedDescriptor.Templates))

	for i, _ := range k.mergedDescriptor.Templates {
		templatePlaceholders[i] = "<generated>"
	}
	kappIntrinsicDataConverted["templates"] = templatePlaceholders

	namespacedKappMap := map[string]interface{}{
		"kapp": kappIntrinsicDataConverted,
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

		// run the source path through the templater in case it contains variables
		templateSource, err := templater.RenderTemplate(rawTemplateSource, templateVars)
		if err != nil {
			return nil, errors.WithStack(err)
		}

		if !filepath.IsAbs(templateSource) {
			foundTemplate := false

			// see whether the template is in the kapp itself
			possibleSource := filepath.Join(k.GetCacheDir(), templateSource)
			log.Logger.Debugf("Searching for kapp template in '%s'", possibleSource)
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
			destPath = filepath.Join(k.GetCacheDir(), destPath)
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
