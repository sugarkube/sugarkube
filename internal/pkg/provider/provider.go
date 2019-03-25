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

package provider

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/sugarkube/sugarkube/internal/pkg/constants"
	"github.com/sugarkube/sugarkube/internal/pkg/interfaces"
	"github.com/sugarkube/sugarkube/internal/pkg/log"
	"github.com/sugarkube/sugarkube/internal/pkg/utils"
	"github.com/sugarkube/sugarkube/internal/pkg/vars"
	"os"
	"path/filepath"
	"strings"
)

// implemented providers
const LOCAL = "local"
const AWS = "aws"

// Factory that creates providers
func newProviderImpl(name string, stackConfig interfaces.IStackConfig) (interfaces.IProvider, error) {
	log.Logger.Debugf("Will try to instantiate a provider called '%s'", name)

	if name == LOCAL {
		return &LocalProvider{}, nil
	}

	if name == AWS {
		return &AwsProvider{
			region: stackConfig.GetRegion(),
		}, nil
	}

	return nil, errors.New(fmt.Sprintf("Provider '%s' doesn't exist", name))
}

// Instantiates a Provider and returns it along with the stack Config vars it can
// load, or an error.
func New(stackConfig interfaces.IStackConfig) (interfaces.IProvider, error) {
	providerImpl, err := newProviderImpl(stackConfig.GetProvider(), stackConfig)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return providerImpl, nil
}

// Return vars loaded from configs that should be passed on to kapps by Installers
func GetInstallerVars(provider interfaces.IProvider) map[string]interface{} {
	return provider.GetInstallerVars()
}

// Returns the name of the provider
func GetName(p interfaces.IProvider) string {
	return p.GetName()
}

// Finds all vars files for the given provider and returns the result of merging
// all the data.
func GetVarsFromFiles(provider interfaces.IProvider,
	stackConfig interfaces.IStackConfig) (map[string]interface{}, error) {
	dirs, err := findVarsFiles(provider, stackConfig)
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

// Search for paths to provider vars files
func findVarsFiles(provider interfaces.IProvider, stackConfig interfaces.IStackConfig) ([]string, error) {
	precedence := []string{
		utils.StripExtension(constants.ValuesFile),
		stackConfig.GetProvider(),
		stackConfig.GetProvisioner(),
		stackConfig.GetAccount(),
		stackConfig.GetProfile(),
		stackConfig.GetCluster(),
		stackConfig.GetRegion(),
	}

	// append the provider-specific static directory names to search
	precedence = append(precedence, provider.CustomVarsDirs()...)

	paths := make([]string, 0)

	log.Logger.Debugf("Provider vars dirs are: %s", strings.Join(stackConfig.GetProviderVarsDirs(), ", "))

	for _, searchDir := range stackConfig.GetProviderVarsDirs() {
		searchPath, err := filepath.Abs(filepath.Join(stackConfig.GetDir(), searchDir))
		if err != nil {
			return nil, errors.WithStack(err)
		}

		log.Logger.Infof("Searching for files/dirs under '%s' with basenames: %s",
			searchPath, strings.Join(precedence, ", "))

		err = utils.PrecedenceWalk(searchPath, precedence, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return errors.WithStack(err)
			}

			log.Logger.Debugf("Walked to path: %s", path)

			if !info.IsDir() {
				ext := filepath.Ext(path)

				if strings.ToLower(ext) != ".yaml" {
					log.Logger.Debugf("Ignoring non-yaml file: %s", path)
					return nil
				}

				log.Logger.Debugf("Adding var file: %s", path)
				paths = append(paths, path)
			}

			return nil
		})

		if err != nil {
			return nil, errors.WithStack(err)
		}
	}

	log.Logger.Debugf("Provider var paths are: %s", strings.Join(paths, ", "))

	return paths, nil
}
