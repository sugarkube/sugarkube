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
	"github.com/sugarkube/sugarkube/internal/pkg/kapp"
	"github.com/sugarkube/sugarkube/internal/pkg/log"
	"github.com/sugarkube/sugarkube/internal/pkg/utils"
	"github.com/sugarkube/sugarkube/internal/pkg/vars"
	"os"
	"path/filepath"
	"strings"
)

type Provider interface {
	// Returns the name of the provider
	getName() string
	// Associate provider variables with the provider
	setVars(map[string]interface{})
	// Returns variables installers should pass on to kapps
	getInstallerVars() map[string]interface{}
	customVarsDirs() []string
}

// implemented providers
const LOCAL = "local"
const AWS = "aws"

// Factory that creates providers
func newProviderImpl(name string, stackConfig *kapp.StackConfig) (Provider, error) {
	if name == LOCAL {
		return &LocalProvider{}, nil
	}

	if name == AWS {
		return &AwsProvider{
			region: stackConfig.Region,
		}, nil
	}

	return nil, errors.New(fmt.Sprintf("Provider '%s' doesn't exist", name))
}

// Instantiates a Provider and returns it along with the stack config vars it can
// load, or an error.
func NewProvider(stackConfig *kapp.StackConfig) (Provider, error) {
	providerImpl, err := newProviderImpl(stackConfig.Provider, stackConfig)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return providerImpl, nil
}

// Return vars loaded from configs that should be passed on to kapps by Installers
func GetInstallerVars(provider Provider) map[string]interface{} {
	return provider.getInstallerVars()
}

// Returns the name of the provider
func GetName(p Provider) string {
	return p.getName()
}

// Finds all vars files for the given provider and returns the result of merging
// all the data.
func GetVarsFromFiles(provider Provider, stackConfig *kapp.StackConfig) (map[string]interface{}, error) {
	dirs, err := findVarsFiles(provider, stackConfig)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	values := map[string]interface{}{}

	err = vars.Merge(&values, dirs...)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return values, nil
}

// Search for paths to provider vars files
func findVarsFiles(provider Provider, stackConfig *kapp.StackConfig) ([]string, error) {
	precedence := []string{
		utils.StripExtension(constants.VALUES_FILE),
		stackConfig.Provider,
		stackConfig.Provisioner,
		stackConfig.Account,
		stackConfig.Profile,
		stackConfig.Cluster,
		stackConfig.Region,
	}

	// append the provider-specific static directory names to search
	precedence = append(precedence, provider.customVarsDirs()...)

	paths := make([]string, 0)

	for _, searchDir := range stackConfig.ProviderVarsDirs {
		searchPath, err := filepath.Abs(filepath.Join(stackConfig.Dir(), searchDir))
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
