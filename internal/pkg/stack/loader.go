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
	"github.com/imdario/mergo"
	"github.com/pkg/errors"
	"github.com/sugarkube/sugarkube/internal/pkg/interfaces"
	"github.com/sugarkube/sugarkube/internal/pkg/log"
	"github.com/sugarkube/sugarkube/internal/pkg/printer"
	"github.com/sugarkube/sugarkube/internal/pkg/provider"
	"github.com/sugarkube/sugarkube/internal/pkg/registry"
	"github.com/sugarkube/sugarkube/internal/pkg/structs"
	"github.com/sugarkube/sugarkube/internal/pkg/utils"
	"gopkg.in/yaml.v2"
	"os"
	"path/filepath"
	"strings"
)

// Loads a stack config from a file. Values are merged with CLI args (which take precedence), and provider
// variables are loaded and set as a property on the stackConfig. So after this step, stackConfig contains
// all config values for the entire stack (although it won't have been templated yet so any '{{var_name}}'
// type strings won't have been interpolated yet.
func BuildStack(stackName string, stackFilePath string, cliStackConfig *structs.StackFile) (interfaces.IStack, error) {

	if strings.TrimSpace(stackName) == "" {
		return nil, errors.New("The stack name is required")
	}

	if strings.TrimSpace(stackFilePath) == "" {
		return nil, errors.New("A stack config file path is required")
	}

	stackFile, err := loadStackFile(stackName, stackFilePath)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	log.Logger.Tracef("Merging raw stack config %#v with CLI values: %#v", stackFile,
		cliStackConfig)

	err = mergo.Merge(stackFile, cliStackConfig, mergo.WithOverride)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	log.Logger.Debugf("Final raw stack file: %#v", stackFile)

	// parse the raw config, populating objects and return a stackConfig
	stackConfig, err := parseStackFile(*stackFile)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	// initialise the provider and load its variables
	providerImpl, err := provider.New(stackConfig)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	registryImpl := registry.New()

	stackObj, err := newStack(stackConfig, providerImpl, registryImpl)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	err = validateStackConfig(stackConfig)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	numKapps := 0
	for _, manifest := range stackConfig.Manifests() {
		numKapps += len(manifest.Installables())
	}

	if stackObj.GetConfig().GetCluster() == "" {
		return nil, errors.New("No cluster name defined. Rerun your command adding `--cluster=<name>` or " +
			"define it in your stack.")
	}

	_, err = printer.Fprintf("[green]Successfully loaded stack config file '%s' containing %d "+
		"manifest(s) and %d kapps in total.\n\n", stackFilePath, len(stackConfig.Manifests()), numKapps)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return stackObj, nil
}

// Loads the config for a stack from a YAML file and returns either it or an error
func loadStackFile(name string, path string) (*structs.StackFile, error) {

	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, errors.Wrap(err, "Can't load non-existent stack file")
	}

	log.Logger.Debugf("Loading stack config from '%s'", path)

	data := map[string]interface{}{}
	err := utils.LoadYamlFile(path, &data)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	stackConfig, ok := data[name]
	if !ok {
		validNames := make([]string, 0)
		for k := range data {
			validNames = append(validNames, k)
		}

		return nil, errors.New(fmt.Sprintf("No stack called '%s' found in stack file '%s'. Valid "+
			"stack names are: %s", name, path, strings.Join(validNames, ", ")))
	}

	log.Logger.Infof("Loaded stack '%s' from file '%s'", name, path)

	// marshal the data we want so we can unmarshal it again into a struct
	stackConfigBytes, err := yaml.Marshal(stackConfig)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	log.Logger.Tracef("Stack config bytes:\n%s", stackConfigBytes)

	// make the path absolute
	path, err = filepath.Abs(path)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	stackObj := structs.StackFile{
		Name:     name,
		FilePath: path,
	}

	err = yaml.Unmarshal(stackConfigBytes, &stackObj)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	log.Logger.Tracef("Loaded raw stack config: %#v", stackObj)

	// at this point the config only contains pointers to manifests
	return &stackObj, nil
}

// Takes a raw config struct and populates the manifests and installables
func parseStackFile(stackFile structs.StackFile) (interfaces.IStackConfig, error) {
	manifests, err := acquireManifests(stackFile)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	stackConfig := &StackConfig{
		stackFile: stackFile,
		manifests: manifests,
	}

	return stackConfig, nil
}
