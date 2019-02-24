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
	"github.com/sugarkube/sugarkube/internal/pkg/provisioner"
	"io"
	"strings"
)

// Loads a stack config from a file
func ProcessCliArgs(stackName string, stackFile string, cliStackConfig *kapp.StackConfig,
	out io.Writer) (*kapp.StackConfig, provider.Provider, provisioner.Provisioner, error) {

	stackConfig, err := LoadStackConfig(stackName, stackFile)
	if err != nil {
		return nil, nil, nil, errors.WithStack(err)
	}

	err = mergo.Merge(stackConfig, cliStackConfig, mergo.WithOverride)
	if err != nil {
		return nil, nil, nil, errors.WithStack(err)
	}

	log.Logger.Debugf("Final stack config: %#v", stackConfig)

	providerImpl, err := provider.NewProvider(stackConfig)
	if err != nil {
		return nil, nil, nil, errors.WithStack(err)
	}

	provisionerImpl, err := provisioner.NewProvisioner(stackConfig.Provisioner)
	if err != nil {
		return nil, nil, nil, errors.WithStack(err)
	}

	err = kapp.ValidateStackConfig(stackConfig)
	if err != nil {
		return nil, nil, nil, errors.WithStack(err)
	}

	numKapps := 0
	for _, manifest := range stackConfig.AllManifests() {
		numKapps += len(manifest.Kapps)
	}

	_, err = fmt.Fprintf(out, "Successfully loaded stack config containing %d "+
		"init manifest(s), %d manifest(s) and %d kapps in total.\n",
		len(stackConfig.InitManifests), len(stackConfig.Manifests), numKapps)
	if err != nil {
		return nil, nil, nil, errors.WithStack(err)
	}

	return stackConfig, providerImpl, provisionerImpl, nil
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
