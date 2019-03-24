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
	descriptor   structs.KappDescriptor
	manifestId   string
	state        string
	config       structs.KappConfig
	rootCacheDir string
}

// Returns the non-fully qualified ID
func (k Kapp) Id() string {
	return k.descriptor.Id
}

// Returns the manifest ID
func (k Kapp) ManifestId() string {
	return k.manifestId
}

func (k Kapp) State() string {
	return k.state
}

func (k Kapp) PostActions() []string {
	return k.descriptor.PostActions
}

func (k Kapp) GetConfig() structs.KappConfig {
	return k.config
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
	return k.config.EnvVars
}

// Return CLI args for the Kapp for the given installer and command/target
func (k Kapp) GetCliArgs(installerName string, command string) map[string]string {
	installerArgs, ok := k.config.Args[installerName]
	if !ok {
		return map[string]string{}
	}

	commandArgs, ok := installerArgs[command]
	if !ok {
		return map[string]string{}
	}

	cliArgs := make(map[string]string, 0)

	for _, arg := range commandArgs {
		strings.Join([]string{arg["name"], arg["value"]}, "=")
	}

	return cliArgs
}

// Sets the root cache directory the kapp is checked out into
func (k *Kapp) SetRootCacheDir(cacheDir string) {
	log.Logger.Debugf("Setting the root cache dir on kapp '%s' to '%s'",
		k.FullyQualifiedId(), cacheDir)
	k.rootCacheDir = cacheDir
}

// todo - this needs a rethink
// Returns the physical path to this kapp in a cache
func (k Kapp) ObjectCacheDir() string {
	cacheDir := filepath.Join(k.rootCacheDir, k.manifestId, k.Id())

	// if no cache dir has been set (e.g. because the user is doing a dry-run),
	// don't return an absolute path
	if k.rootCacheDir != "" {
		absCacheDir, err := filepath.Abs(cacheDir)
		if err != nil {
			panic(fmt.Sprintf("Couldn't convert path to absolute path: %#v", err))
		}

		cacheDir = absCacheDir
	} else {
		log.Logger.Debug("No cache dir has been set on kapp. Cache dir will " +
			"not be converted to an absolute path.")
	}

	return cacheDir
}

// Returns an array of acquirers configured for the sources for this kapp. We need to recompute these each time
// instead of caching them so that any manifest overrides will take effect.
func (k Kapp) Acquirers() ([]acquirer.Acquirer, error) {
	acquirers, err := acquirer.GetAcquirersFromSources(k.descriptor.Sources)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return acquirers, nil
}

// (Re)loads the kapp's sugarkube.yaml file, templates it and saves is at as an attribute on the kapp
func (k *Kapp) RefreshConfig(templateVars map[string]interface{}) error {

	configFilePaths, err := utils.FindFilesByPattern(k.ObjectCacheDir(), constants.KappConfigFileName,
		true, false)
	if err != nil {
		return errors.Wrapf(err, "Error finding '%s' in '%s'",
			constants.KappConfigFileName, k.ObjectCacheDir())
	}

	if len(configFilePaths) == 0 {
		return errors.New(fmt.Sprintf("No '%s' file found for kapp "+
			"'%s' in %s", constants.KappConfigFileName, k.FullyQualifiedId(), k.ObjectCacheDir()))
	} else if len(configFilePaths) > 1 {
		// todo - have a way of declaring the 'right' one in the manifest
		panic(fmt.Sprintf("Multiple '%s' found for kapp '%s'. Disambiguation "+
			"not implemented yet: %s", constants.KappConfigFileName, k.FullyQualifiedId(),
			strings.Join(configFilePaths, ", ")))
	}

	configFilePath := configFilePaths[0]

	var outBuf bytes.Buffer
	err = templater.TemplateFile(configFilePath, &outBuf, templateVars)
	if err != nil {
		return errors.WithStack(err)
	}

	log.Logger.Tracef("Rendered %s file at '%s' to: \n%s", constants.KappConfigFileName,
		configFilePath, outBuf.String())

	config := structs.KappConfig{}
	err = yaml.Unmarshal(outBuf.Bytes(), &config)
	if err != nil {
		return errors.Wrapf(err, "Error unmarshalling rendered %s file: %s",
			constants.KappConfigFileName, outBuf.String())
	}

	k.config = config
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
	err = mergo.Merge(&kappVars, k.descriptor.Vars, mergo.WithOverride)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	// namespace kapp variables. This is safer than letting kapp variables overwrite arbitrary values (e.g.
	// so they can't change the target stack, whether the kapp's approved, etc.), but may be too restrictive
	// in certain situations. We'll have to see
	kappIntrinsicDataConverted["vars"] = kappVars

	// add placeholders templated paths so kapps that use them work when running
	// `kapp vars`, etc.
	templatePlaceholders := make([]string, len(k.descriptor.Templates))

	for i, _ := range k.descriptor.Templates {
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
		"cacheRoot": k.ObjectCacheDir(),
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
		stackConfig.Name(),
		stackConfig.Provider(),
		stackConfig.Provisioner(),
		stackConfig.Account(),
		stackConfig.Region(),
		stackConfig.Profile(),
		stackConfig.Cluster(),
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
		searchPath, err := filepath.Abs(filepath.Join(stackConfig.Dir(), searchDir))
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

	renderedPaths := make([]string, 0)

	if len(k.descriptor.Templates) == 0 {
		log.Logger.Infof("No templates to render for kapp '%s'", k.FullyQualifiedId())
		return renderedPaths, nil
	}

	log.Logger.Infof("Rendering templates for kapp '%s'", k.FullyQualifiedId())

	for _, templateDefinition := range k.descriptor.Templates {
		rawTemplateSource := templateDefinition.Source

		// run the source path through the templater in case it contains variables
		templateSource, err := templater.RenderTemplate(rawTemplateSource, templateVars)
		if err != nil {
			return nil, errors.WithStack(err)
		}

		if !filepath.IsAbs(templateSource) {
			foundTemplate := false

			// see whether the template is in the kapp itself
			possibleSource := filepath.Join(k.ObjectCacheDir(), templateSource)
			log.Logger.Debugf("Searching for kapp template in '%s'", possibleSource)
			_, err := os.Stat(possibleSource)
			if err == nil {
				templateSource = possibleSource
				foundTemplate = true
			}

			if !foundTemplate {
				// search each template directory defined in the stack config
				for _, templateDir := range stackConfig.TemplateDirs() {
					possibleSource := filepath.Join(stackConfig.Dir(), templateDir, templateSource)
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

		log.Logger.Debugf("Templating file '%s' with vars: %#v", templateSource, templateVars)

		rawDestPath := templateDefinition.Dest
		// run the dest path through the templater in case it contains variables
		destPath, err := templater.RenderTemplate(rawDestPath, templateVars)
		if err != nil {
			return nil, errors.WithStack(err)
		}

		if !filepath.IsAbs(destPath) {
			destPath = filepath.Join(k.ObjectCacheDir(), destPath)
		}

		// check whether the dest path exists
		if _, err := os.Stat(destPath); err == nil {
			log.Logger.Infof("Template destination path '%s' exists. "+
				"File will be overwritten by rendered template '%s' for kapp '%s'",
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

		if dryRun {
			log.Logger.Infof("Dry run. Template '%s' for kapp '%s' which "+
				"would be written to '%s' rendered as:\n%s", templateSource,
				k.Id(), destPath, outBuf.String())
		} else {
			log.Logger.Infof("Writing rendered template '%s' for kapp "+
				"'%s' to '%s'", templateSource, k.FullyQualifiedId(), destPath)
			err := ioutil.WriteFile(destPath, outBuf.Bytes(), 0644)
			if err != nil {
				return renderedPaths, errors.WithStack(err)
			}
		}

		renderedPaths = append(renderedPaths, destPath)
	}

	return renderedPaths, nil
}
