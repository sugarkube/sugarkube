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
	"bytes"
	"fmt"
	"github.com/pkg/errors"
	"github.com/sugarkube/sugarkube/internal/pkg/kapp"
	"github.com/sugarkube/sugarkube/internal/pkg/log"
	"github.com/sugarkube/sugarkube/internal/pkg/provider"
	"github.com/sugarkube/sugarkube/internal/pkg/utils"
	"os"
	"path/filepath"
	"strings"
)

// Installs kapps with make
type MakeInstaller struct {
	provider        provider.Provider
	stackConfigVars provider.Values
}

const TARGET_INSTALL = "install"
const TARGET_DESTROY = "destroy"

// Run the given make target
func (i MakeInstaller) run(makeTarget string, kappObj *kapp.Kapp,
	stackConfig *kapp.StackConfig, approved bool, dryRun bool) error {

	// search for the Makefile
	makefilePaths, err := findFilesByPattern(kappObj.RootDir, "Makefile",
		true, false)
	if err != nil {
		return errors.Wrapf(err, "Error finding Makefile in '%s'",
			kappObj.RootDir)
	}

	if len(makefilePaths) == 0 {
		return errors.New(fmt.Sprintf("No makefile found for kapp '%s' "+
			"in '%s'", kappObj.Id, kappObj.RootDir))
	}
	if len(makefilePaths) > 1 {
		// todo - select the right makefile from the installerConfig if it exists,
		// then remove this panic
		panic(fmt.Sprintf("Multiple Makefiles found. Disambiguation "+
			"not implemented yet: %s", strings.Join(makefilePaths, ", ")))
	}

	absKappRoot, err := filepath.Abs(kappObj.RootDir)
	if err != nil {
		return errors.WithStack(err)
	}

	// create the env vars
	envVars := map[string]string{
		"KAPP_ROOT": absKappRoot,
		"APPROVED":  fmt.Sprintf("%v", approved),
		"CLUSTER":   stackConfig.Cluster,
		"PROFILE":   stackConfig.Profile,
		"PROVIDER":  stackConfig.Provider,
	}

	providerImpl, err := provider.NewProvider(stackConfig)
	if err != nil {
		return errors.WithStack(err)
	}

	parameterisers, err := identifyKappInterfaces(kappObj)
	if err != nil {
		return errors.WithStack(err)
	}

	// Adds things like `KUBE_CONTEXT`, `NAMESPACE`, `RELEASE`, etc.
	for _, parameteriser := range parameterisers {
		pEnvVars, err := parameteriser.GetEnvVars(provider.GetVars(providerImpl))
		if err != nil {
			return errors.WithStack(err)
		}

		for k, v := range pEnvVars {
			envVars[k] = v
		}
	}

	// Provider-specific env vars, e.g. the AwsProvider adds REGION
	for k, v := range provider.GetInstallerVars(i.provider) {
		upperKey := strings.ToUpper(k)
		envVars[upperKey] = fmt.Sprintf("%#v", v)
	}

	// add our env vars to the user's existing env vars
	strEnvVars := os.Environ()
	for k, v := range envVars {
		strEnvVars = append(strEnvVars, strings.Join([]string{k, v}, "="))
	}

	// get additional CLI args
	validPatternMatches := []string{
		stackConfig.Cluster,
		stackConfig.Profile,
		stackConfig.Provider,
	}

	cliArgs := []string{makeTarget}
	for _, parameteriser := range parameterisers {
		arg, err := parameteriser.GetCliArgs(validPatternMatches)
		if err != nil {
			return errors.WithStack(err)
		}

		if arg != "" {
			cliArgs = append(cliArgs, arg)
		}
	}

	makefilePath, err := filepath.Abs(makefilePaths[0])
	if err != nil {
		return errors.WithStack(err)
	}

	log.Infof("Installing kapp '%s'...", kappObj.Id)

	var stdoutBuf, stderrBuf bytes.Buffer
	err = utils.ExecCommand("make", cliArgs, &stdoutBuf, &stderrBuf,
		filepath.Dir(makefilePath), 0, dryRun)
	if err != nil {
		return errors.WithStack(err)
	}

	log.Infof("Kapp '%s' successfully %sed", kappObj.Id, makeTarget)

	return nil
}

// Install a kapp
func (i MakeInstaller) install(kappObj *kapp.Kapp, stackConfig *kapp.StackConfig,
	approved bool, dryRun bool) error {
	return i.run(TARGET_INSTALL, kappObj, stackConfig, approved, dryRun)
}

// Destroy a kapp
func (i MakeInstaller) destroy(kappObj *kapp.Kapp, stackConfig *kapp.StackConfig,
	approved bool, dryRun bool) error {
	return i.run(TARGET_DESTROY, kappObj, stackConfig, approved, dryRun)
}
