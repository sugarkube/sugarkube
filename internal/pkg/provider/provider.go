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
	"github.com/sugarkube/sugarkube/internal/pkg/vars"
	"os"
	"path/filepath"
)

type Provider interface {
	// Returns the name of the provider
	getName() string
	// Returns the variables loaded by the Provider
	getVars() map[string]interface{}
	// Associate provider variables with the provider
	setVars(map[string]interface{})
	// Returns variables installers should pass on to kapps
	getInstallerVars() map[string]interface{}
	// Method that returns all paths in a config directory relevant to the
	// target profile/cluster/region, etc. that should be searched for values
	// files to merge.
	varsDirs(sc *kapp.StackConfig) ([]string, error)
}

// implemented providers
const LOCAL = "local"
const AWS = "aws"

// Factory that creates providers
func newProviderImpl(name string) (Provider, error) {
	if name == LOCAL {
		return &LocalProvider{}, nil
	}

	if name == AWS {
		return &AwsProvider{}, nil
	}

	return nil, errors.New(fmt.Sprintf("Provider '%s' doesn't exist", name))
}

// Instantiates a Provider and returns it along with the stack config vars it can
// load, or an error.
func NewProvider(stackConfig *kapp.StackConfig) (Provider, error) {
	providerImpl, err := newProviderImpl(stackConfig.Provider)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	stackConfigVars, err := stackConfigVars(providerImpl, stackConfig)
	if err != nil {
		log.Logger.Warn("Error loading stack config variables")
		return nil, errors.WithStack(err)
	}
	log.Logger.Debugf("Provider loaded vars: %#v", stackConfigVars)

	if len(stackConfigVars) == 0 {
		log.Logger.Fatal("No values loaded for stack")
		return nil, errors.New("Failed to load values for stack")
	}

	providerImpl.setVars(stackConfigVars)

	return providerImpl, nil
}

// Searches for values.yaml files in configured directories and returns the
// result of merging them.
func stackConfigVars(p Provider, sc *kapp.StackConfig) (map[string]interface{}, error) {
	stackConfigVars := map[string]interface{}{}

	varsDirs, err := p.varsDirs(sc)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	for _, varFile := range varsDirs {
		valuePath := filepath.Join(varFile, constants.VALUES_FILE)

		_, err := os.Stat(valuePath)
		if err != nil {
			log.Logger.Debugf("Skipping merging non-existent path %s", valuePath)
			continue
		}

		err = vars.Merge(&stackConfigVars, valuePath)
		if err != nil {
			return nil, errors.WithStack(err)
		}
	}

	return stackConfigVars, nil
}

// Return vars loaded from configs
func GetVars(p Provider) map[string]interface{} {
	return p.getVars()
}

// Return vars loaded from configs that should be passed on to kapps by Installers
func GetInstallerVars(p Provider) map[string]interface{} {
	return p.getInstallerVars()
}

// Returns the name of the provider
func GetName(p Provider) string {
	return p.getName()
}
