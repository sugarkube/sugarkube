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
	"github.com/sugarkube/sugarkube/internal/pkg/kapp"
	"github.com/sugarkube/sugarkube/internal/pkg/log"
	"github.com/sugarkube/sugarkube/internal/pkg/provider"
	"github.com/sugarkube/sugarkube/internal/pkg/registry"
	"github.com/sugarkube/sugarkube/internal/pkg/structs"
	"io"
	"strings"
)

// This is in a different package to the Stack struct to avoid circular dependencies
// in packages that use it.

// Loads a stack config from a file. Values are merged with CLI args (which take precedence), and provider
// variables are loaded and set as a property on the stackConfig. So after this step, stackConfig contains
// all config values for the entire stack (although it won't have been templated yet so any '{{var_name}}'
// type strings won't have been interpolated yet.
func BuildStack(stackName string, stackFile string, cliStackConfig *kapp.StackConfig,
	out io.Writer) (*structs.Stack, error) {

	if strings.TrimSpace(stackName) == "" {
		return nil, errors.New("The stack name is required")
	}

	if strings.TrimSpace(stackFile) == "" {
		return nil, errors.New("A stack config file path is required")
	}

	stackConfig, err := kapp.LoadStackConfig(stackName, stackFile)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	log.Logger.Debugf("Parsed stack CLI args to stack config: %#v", stackConfig)

	err = mergo.Merge(stackConfig, cliStackConfig, mergo.WithOverride)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	log.Logger.Debugf("Final stack config: %#v", stackConfig)

	// initialise the provider and load its variables
	providerImpl, err := provider.NewProvider(stackConfig)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	providerVars, err := provider.GetVarsFromFiles(providerImpl, stackConfig)
	if err != nil {
		log.Logger.Warn("Error loading provider variables")
		return nil, errors.WithStack(err)
	}
	log.Logger.Debugf("Provider loaded vars: %#v", providerVars)

	if len(providerVars) == 0 {
		log.Logger.Fatal("No values loaded for provider")
		return nil, errors.New(fmt.Sprintf("Failed to load variables for provider %s",
			provider.GetName(providerImpl)))
	}

	stackConfig.SetProviderVars(providerVars)

	registryImpl := registry.NewRegistry()

	stackObj, err := structs.NewStack(stackConfig, providerImpl, &registryImpl)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	err = kapp.ValidateStackConfig(stackConfig)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	numKapps := 0
	for _, manifest := range stackConfig.Manifests {
		numKapps += len(manifest.ParsedKapps())
	}

	_, err = fmt.Fprintf(out, "Successfully loaded stack config containing %d "+
		"manifest(s) and %d kapps in total.\n", len(stackConfig.Manifests), numKapps)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return stackObj, nil
}
