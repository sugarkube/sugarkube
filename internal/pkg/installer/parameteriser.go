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

package installer

import (
	"fmt"
	"github.com/sugarkube/sugarkube/internal/pkg/kapp"
	"github.com/sugarkube/sugarkube/internal/pkg/log"
	"github.com/sugarkube/sugarkube/internal/pkg/provider"
	"os"
	"path/filepath"
	"strings"
)

const IMPLEMENTS_HELM = "helm"
const IMPLEMENTS_TERRAFORM = "terraform"

type Parameteriser struct {
	Name         string
	kappObj      *kapp.Kapp
	providerImpl *provider.Provider
}

// Returns a list of args that the installer should pass to the kapp. This will
// need refactoring once parsing the Parameteriser config is implemented.
func (p *Parameteriser) GetCliArgs(configSubstrings []string) (string, error) {
	filenameTemplate := ""
	argName := ""
	argKey := ""

	if p.Name == IMPLEMENTS_HELM {
		filenameTemplate = "values-{substring}.yaml"
		argName = "helm-opts"
		argKey = "-f"
	}

	log.Logger.Debugf("Building CLI args for the '%s' parameteriser", p.Name)

	// todo - only do this if approved=false. Terraform won't let us pass parameters when
	//  applying a plan
	if p.Name == IMPLEMENTS_TERRAFORM {
		providerName := provider.GetName(*p.providerImpl)
		terraformDir := fmt.Sprintf("terraform_%s", strings.ToLower(providerName))
		filenameTemplate = filepath.Join(terraformDir, "vars", "{substring}.tfvars")
		argName = "tf-opts"
		argKey = "-var-file"
	}

	if filenameTemplate == "" {
		return "", nil
	}

	// todo - support defaults.yaml or defaults.tfvars
	cliValues := []string{}
	seenPaths := map[string]bool{}

	// if the file exists, add it to the list of CLI args
	for _, substring := range configSubstrings {
		filename := strings.Replace(filenameTemplate, "{substring}", substring, 1)

		// iterate through all kapp sources
		for _, kappAcquirer := range p.kappObj.Sources {
			if !kappAcquirer.IncludeValues() {
				log.Logger.Debugf("Won't search kapp source '%s' for values files",
					kappAcquirer.Name())
				continue
			}

			path := filepath.Join(p.kappObj.CacheDir(), kappAcquirer.Name(), filename)

			// ignore paths we've already seen
			if _, ok := seenPaths[path]; ok {
				continue
			}

			if _, err := os.Stat(path); err == nil {
				arg := strings.Join([]string{argKey, path}, " ")
				cliValues = append(cliValues, arg)
				seenPaths[path] = true
			}
		}
	}

	cliArg := ""

	if len(cliValues) > 0 {
		joinedValues := strings.Join(cliValues, " ")
		cliArg = strings.Join([]string{argName, joinedValues}, "=")
	}

	log.Logger.Debugf("Returning CLI arg for kapp %s: %s",
		p.kappObj.FullyQualifiedId(), cliArg)

	return cliArg, nil
}
