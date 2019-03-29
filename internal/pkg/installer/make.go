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
	"github.com/imdario/mergo"
	"github.com/pkg/errors"
	"github.com/sugarkube/sugarkube/internal/pkg/interfaces"
	"github.com/sugarkube/sugarkube/internal/pkg/log"
	"github.com/sugarkube/sugarkube/internal/pkg/provider"
	"github.com/sugarkube/sugarkube/internal/pkg/utils"
	"path/filepath"
	"strings"
)

// Installs kapps with make
type MakeInstaller struct {
	provider interfaces.IProvider
}

const TargetInstall = "install"
const TargetDelete = "delete"

// Return the name of this installer
func (i MakeInstaller) name() string {
	return "make"
}

// Run the given make target
func (i MakeInstaller) run(makeTarget string, installable interfaces.IInstallable, stack interfaces.IStack,
	approved bool, renderTemplates bool, dryRun bool) error {

	// search for the Makefile
	makefilePaths, err := utils.FindFilesByPattern(installable.ObjectCacheDir(), "Makefile",
		true, false)
	if err != nil {
		return errors.Wrapf(err, "Error finding Makefile in '%s'",
			installable.ObjectCacheDir())
	}

	if len(makefilePaths) == 0 {
		return errors.New(fmt.Sprintf("No makefile found for kapp '%s' "+
			"in '%s'", installable.Id(), installable.ObjectCacheDir()))
	}
	if len(makefilePaths) > 1 {
		// todo - select the right makefile from the installerConfig if it exists,
		// then remove this panic
		panic(fmt.Sprintf("Multiple Makefiles found. Disambiguation "+
			"not implemented yet: %s", strings.Join(makefilePaths, ", ")))
	}

	// merge all the vars required to render the kapp's sugarkube.yaml file
	templatedVars, err := stack.GetTemplatedVars(installable,
		map[string]interface{}{"target": makeTarget, "approved": approved})

	if renderTemplates {
		renderedTemplates, err := installable.RenderTemplates(templatedVars, stack.GetConfig(), dryRun)
		if err != nil {
			return errors.WithStack(err)
		}

		// merge renderedTemplates into the templatedVars under the "kapp.templates" key. This will
		// allow us to support writing files to temporary (dynamic) locations later if we like
		renderedTemplatesMap := map[string]interface{}{
			"kapp": map[string]interface{}{
				"templates": renderedTemplates,
			},
		}

		log.Logger.Debugf("Merging rendered template paths into stack config: %#v",
			renderedTemplates)

		err = mergo.Merge(&templatedVars, renderedTemplatesMap, mergo.WithOverride)
		if err != nil {
			return errors.WithStack(err)
		}
	} else {
		log.Logger.Infof("Skipping writing templates for kapp '%s'", installable.FullyQualifiedId())
	}

	// load the kapp's own config
	err = installable.RefreshConfig(templatedVars)
	if err != nil {
		return errors.WithStack(err)
	}

	stackConfig := stack.GetConfig()

	// populate env vars that are always supplied
	envVars := map[string]string{
		"KAPP_ROOT": installable.ObjectCacheDir(),
		"APPROVED":  fmt.Sprintf("%v", approved),
		"CLUSTER":   stackConfig.GetCluster(),
		"PROFILE":   stackConfig.GetProfile(),
		"PROVIDER":  stackConfig.GetProvider(),
	}

	// Provider-specific env vars, e.g. the AwsProvider adds REGION
	for k, v := range provider.GetInstallerVars(i.provider) {
		upperKey := strings.ToUpper(k)
		envVars[upperKey] = fmt.Sprintf("%#v", v)
	}

	// add the env vars the kapp needs
	for k, v := range installable.GetEnvVars() {
		upperKey := strings.ToUpper(k)
		envVars[upperKey] = strings.Trim(fmt.Sprintf("%#v", v), "\"")
	}

	cliArgs := []string{makeTarget}

	// todo - merge in values from the global config for each program declared in
	// the kapp's sugarkube.yaml file. Make sure to respect the registry...

	// todo - move this to a method. Make it pull values from
	// the overall config depending on the programs the kapp uses
	targetArgs := installable.GetCliArgs(i.name(), makeTarget)
	log.Logger.Debugf("Kapp '%s' has args for %s %s (approved=%v): %#v",
		installable.FullyQualifiedId(), i.name(), makeTarget, approved, targetArgs)

	for _, targetArg := range targetArgs {
		cliArgs = append(cliArgs, targetArg)
	}

	makefilePath, err := filepath.Abs(makefilePaths[0])
	if err != nil {
		return errors.WithStack(err)
	}

	if approved {
		log.Logger.Infof("Installing kapp '%s'...", installable.FullyQualifiedId())
	} else {
		log.Logger.Infof("Planning kapp '%s'...", installable.FullyQualifiedId())
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

	log.Logger.Infof("Kapp '%s' successfully processed", installable.FullyQualifiedId())

	return nil
}

// Install a kapp
func (i MakeInstaller) install(installableObj interfaces.IInstallable, stack interfaces.IStack,
	approved bool, renderTemplates bool, dryRun bool) error {
	return i.run(TargetInstall, installableObj, stack, approved, renderTemplates, dryRun)
}

// Delete a kapp
func (i MakeInstaller) delete(installableObj interfaces.IInstallable, stack interfaces.IStack,
	approved bool, renderTemplates bool, dryRun bool) error {
	return i.run(TargetDelete, installableObj, stack, approved, renderTemplates, dryRun)
}
