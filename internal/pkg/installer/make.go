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
	"github.com/sugarkube/sugarkube/internal/pkg/constants"
	"github.com/sugarkube/sugarkube/internal/pkg/interfaces"
	"github.com/sugarkube/sugarkube/internal/pkg/log"
	"github.com/sugarkube/sugarkube/internal/pkg/printer"
	"github.com/sugarkube/sugarkube/internal/pkg/utils"
	"path/filepath"
	"strings"
)

// Installs kapps with make
type MakeInstaller struct {
	provider interfaces.IProvider
}

const TargetInstall = "install" // todo - replace this with 2 targets - plan-install and apply-install
const TargetDelete = "delete"   // todo - replace this with 2 targets - plan-delete and apply-delete
const TargetOutput = "output"
const TargetClean = "clean"

// Return the name of this installer
func (i MakeInstaller) Name() string {
	return Make
}

// Run the given make target
func (i MakeInstaller) run(makeTarget string, installable interfaces.IInstallable, stack interfaces.IStack,
	approved bool, dryRun bool) error {

	// search for the Makefile
	makefilePaths, err := utils.FindFilesByPattern(installable.GetCacheDir(), "Makefile",
		true, false)
	if err != nil {
		return errors.Wrapf(err, "Error finding Makefile in '%s'",
			installable.GetCacheDir())
	}

	if len(makefilePaths) == 0 {
		return errors.New(fmt.Sprintf("No makefile found for kapp '%s' "+
			"in '%s'", installable.Id(), installable.GetCacheDir()))
	}
	if len(makefilePaths) > 1 {
		// todo - select the right makefile from the installerConfig if it exists,
		// then remove this panic
		panic(fmt.Sprintf("Multiple Makefiles found. Disambiguation "+
			"not implemented yet: %s", strings.Join(makefilePaths, ", ")))
	}

	stackConfig := stack.GetConfig()

	// populate env vars that are always supplied
	envVars := map[string]string{
		"KAPP_ROOT": installable.GetCacheDir(),
		"APPROVED":  fmt.Sprintf("%v", approved),
		"CLUSTER":   stackConfig.GetCluster(),
		"PROFILE":   stackConfig.GetProfile(),
		"PROVIDER":  stackConfig.GetProvider(),
	}

	// Provider-specific env vars, e.g. the AwsProvider adds REGION
	for k, v := range i.provider.GetInstallerVars() {
		upperKey := strings.ToUpper(k)
		envVars[upperKey] = fmt.Sprintf("%#v", v)
	}

	// add all kapp vars as env vars
	installableVars, err := installable.Vars(stack)
	if err != nil {
		return errors.WithStack(err)
	}

	kappAllVars, ok := installableVars[constants.KappVarsKappKey]
	if ok {
		kappAllVarsMap := kappAllVars.(map[string]interface{})
		kappVars, stillOk := kappAllVarsMap[constants.KappVarsVarsKey]
		if stillOk {
			kappVarsMap := kappVars.(map[string]interface{})
			for k, v := range kappVarsMap {
				upperKey := strings.ToUpper(k)
				envVars[upperKey] = strings.Trim(fmt.Sprintf("%#v", v), "\"")
			}
		}
	}

	// now add explicitly defined env vars
	for k, v := range installable.GetEnvVars() {
		upperKey := strings.ToUpper(k)
		envVars[upperKey] = strings.Trim(fmt.Sprintf("%#v", v), "\"")
	}

	cliArgs := []string{makeTarget}

	targetArgs := installable.GetCliArgs(i.Name(), makeTarget)
	log.Logger.Debugf("Kapp '%s' has args for %s %s (approved=%v): %#v",
		installable.FullyQualifiedId(), i.Name(), makeTarget, approved, targetArgs)

	for _, targetArg := range targetArgs {
		cliArgs = append(cliArgs, targetArg)
	}

	makefilePath, err := filepath.Abs(makefilePaths[0])
	if err != nil {
		return errors.WithStack(err)
	}

	_, err = printer.Fprintf("Running '[white]make %s[default]' on kapp '[white]%s[default]' with APPROVED=%v...\n", makeTarget,
		installable.FullyQualifiedId(), approved)
	if err != nil {
		return errors.WithStack(err)
	}

	var stdoutBuf, stderrBuf bytes.Buffer
	err = utils.ExecCommand("make", cliArgs, envVars, &stdoutBuf,
		&stderrBuf, filepath.Dir(makefilePath), 0, dryRun)

	log.Logger.Infof("Stdout: %s", stdoutBuf.String())
	log.Logger.Infof("Stderr: %s", stderrBuf.String())

	// some commands write to stderr, so we can't just fail if that buffer is non-zero
	if err != nil {
		return errors.WithStack(err)
	}

	log.Logger.Infof("Kapp '%s' successfully processed (approved=%v, dry run=%v)",
		installable.FullyQualifiedId(), approved, dryRun)

	_, err = printer.Fprintf("[green]Successfully ran[default] '[white]make %s[default]' on "+
		"[default]'[white]%s[default]' with [bold]APPROVED=%v\n",
		makeTarget, installable.FullyQualifiedId(), approved)
	if err != nil {
		return errors.WithStack(err)
	}

	return nil
}

// Plan installation of a kapp
func (i MakeInstaller) PlanInstall(installableObj interfaces.IInstallable, stack interfaces.IStack,
	dryRun bool) error {
	log.Logger.Infof("Planning installation of kapp '%s' (dry run=%v)...",
		installableObj.FullyQualifiedId(), dryRun)
	return i.run(TargetInstall, installableObj, stack, false, dryRun)
}

// Actually install a kapp
func (i MakeInstaller) ApplyInstall(installableObj interfaces.IInstallable, stack interfaces.IStack,
	dryRun bool) error {
	log.Logger.Infof("Installing kapp '%s' (dry run=%v)...",
		installableObj.FullyQualifiedId(), dryRun)
	return i.run(TargetInstall, installableObj, stack, true, dryRun)
}

// Plan deletion of a kapp
func (i MakeInstaller) PlanDelete(installableObj interfaces.IInstallable, stack interfaces.IStack,
	dryRun bool) error {
	log.Logger.Infof("Planning deletion of kapp '%s' (dry run=%v)...",
		installableObj.FullyQualifiedId(), dryRun)
	return i.run(TargetDelete, installableObj, stack, false, dryRun)
}

// Actually delete a kapp
func (i MakeInstaller) ApplyDelete(installableObj interfaces.IInstallable, stack interfaces.IStack,
	dryRun bool) error {
	log.Logger.Infof("Deleting kapp '%s' (dry run=%v)...",
		installableObj.FullyQualifiedId(), dryRun)
	return i.run(TargetDelete, installableObj, stack, true, dryRun)
}

// Get a kapp's outputs
func (i MakeInstaller) Output(installableObj interfaces.IInstallable, stack interfaces.IStack,
	dryRun bool) error {
	log.Logger.Infof("Getting output for kapp '%s'...", installableObj.FullyQualifiedId())
	return i.run(TargetOutput, installableObj, stack, true, dryRun)
}

// Clean a kapp
func (i MakeInstaller) Clean(installableObj interfaces.IInstallable, stack interfaces.IStack,
	dryRun bool) error {
	log.Logger.Infof("Cleaning kapp '%s'...", installableObj.FullyQualifiedId())
	return i.run(TargetClean, installableObj, stack, true, dryRun)
}

func (i MakeInstaller) GetVars(action string, approved bool) map[string]interface{} {
	return map[string]interface{}{
		"action":   action,
		"approved": fmt.Sprintf("%v", approved)}
}
