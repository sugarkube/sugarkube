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

package utils

import (
	"fmt"
	"github.com/imdario/mergo"
	"github.com/pkg/errors"
	"github.com/sugarkube/sugarkube/internal/pkg/kapp"
	"github.com/sugarkube/sugarkube/internal/pkg/log"
	"github.com/sugarkube/sugarkube/internal/pkg/provider"
	"io"
	"strings"
)

// Loads a stack config from a file. Values are merged with CLI args (which take precedence), and provider
// variables are loaded and set as a property on the stackConfig. So after this step, stackConfig contains
// all config values for the entire stack (although it won't have been templated yet so any '{{var_name}}'
// type strings won't have been interpolated yet.
func BuildStackConfig(stackName string, stackFile string, cliStackConfig *kapp.StackConfig,
	out io.Writer) (*kapp.StackConfig, error) {

	stackConfig, err := LoadStackConfig(stackName, stackFile)
	if err != nil {
		return nil, errors.WithStack(err)
	}

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

	providerVars, err := provider.LoadProviderVars(providerImpl, stackConfig)
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

	err = kapp.ValidateStackConfig(stackConfig)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	numKapps := 0
	for _, manifest := range stackConfig.AllManifests() {
		numKapps += len(manifest.ParsedKapps())
	}

	_, err = fmt.Fprintf(out, "Successfully loaded stack config containing %d "+
		"init manifest(s), %d manifest(s) and %d kapps in total.\n",
		len(stackConfig.InitManifests), len(stackConfig.Manifests), numKapps)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return stackConfig, nil
}

// Loads a named stack from a stack config file or returns an error
func LoadStackConfig(stackName string, stackFile string) (*kapp.StackConfig, error) {

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

	return stackConfig, nil
}

// Return kapps selected by inclusion/exclusion selectors from the given manifests
func SelectKapps(manifests []*kapp.Manifest, includeSelector []string, excludeSelector []string) (map[string]kapp.Kapp, error) {
	selectedKapps := map[string]kapp.Kapp{}
	var err error

	if len(includeSelector) > 0 {
		selectedKapps, err = kapp.GetKappsBySelector(includeSelector, manifests)
		if err != nil {
			return nil, errors.WithStack(err)
		}
		log.Logger.Debugf("Adding %d kapps to the candidate template set", len(selectedKapps))
	} else {
		log.Logger.Debugf("Adding all kapps to the candidate template set")

		log.Logger.Debugf("Stack config has %d manifests", len(manifests))

		// select all kapps
		for _, manifest := range manifests {
			log.Logger.Debugf("Manifest '%s' contains %d kapps", manifest.Id(), len(manifest.ParsedKapps()))

			for _, manifestKapp := range manifest.ParsedKapps() {
				selectedKapps[manifestKapp.FullyQualifiedId()] = manifestKapp
			}
		}
	}

	log.Logger.Debugf("There are %d candidate kapps (before applying exclusions)", len(selectedKapps))

	if len(excludeSelector) > 0 {
		// delete kapps
		excludedKapps, err := kapp.GetKappsBySelector(excludeSelector, manifests)
		if err != nil {
			return nil, errors.WithStack(err)
		}

		log.Logger.Debugf("Excluding %d kapps from the templating set", len(excludedKapps))

		for k := range excludedKapps {
			if _, ok := selectedKapps[k]; ok {
				delete(selectedKapps, k)
			}
		}
	}

	log.Logger.Infof("%d kapps selected for processing in total", len(selectedKapps))

	return selectedKapps, nil
}
