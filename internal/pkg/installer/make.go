package installer

import (
	"bytes"
	"fmt"
	"github.com/pkg/errors"
	"github.com/sugarkube/sugarkube/internal/pkg/kapp"
	"github.com/sugarkube/sugarkube/internal/pkg/log"
	"github.com/sugarkube/sugarkube/internal/pkg/provider"
	"os/exec"
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
		true)
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

	makefilePath := makefilePaths[0]

	// create the env vars
	envVars := map[string]string{
		"APPROVED": fmt.Sprintf("%v", approved),
		"CLUSTER":  stackConfig.Cluster,
		"PROFILE":  stackConfig.Profile,
		"PROVIDER": stackConfig.Provider,
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
		for k, v := range parameteriser.GetEnvVars(provider.GetVars(providerImpl)) {
			envVars[k] = v
		}
	}

	// Provider-specific env vars, e.g. the AwsProvider adds REGION
	for k, v := range provider.GetInstallerVars(i.provider) {
		upperKey := strings.ToUpper(k)
		envVars[upperKey] = fmt.Sprintf("%#v", v)
	}

	// convert the env vars to a string array
	strEnvVars := make([]string, 0)
	for k, v := range envVars {
		strEnvVars = append(strEnvVars, strings.Join([]string{k, v}, "="))
	}

	// get additional CLI args
	validPatternMatches := []string{
		stackConfig.Cluster,
		stackConfig.Profile,
		stackConfig.Provider,
	}

	cliArgs := make([]string, 0)
	for _, parameteriser := range parameterisers {
		arg, err := parameteriser.GetCliArgs(validPatternMatches)
		if err != nil {
			return errors.WithStack(err)
		}

		if arg != "" {
			cliArgs = append(cliArgs, arg)
		}
	}

	// build the command
	var stderrBuf bytes.Buffer

	// make command
	makeCmd := exec.Command(makefilePath, TARGET_INSTALL)
	makeCmd.Dir = filepath.Dir(makefilePath)
	makeCmd.Env = strEnvVars
	makeCmd.Args = cliArgs
	makeCmd.Stderr = &stderrBuf

	if dryRun {
		log.Infof("Dry run. Would install kapp '%s' with command: %s %s",
			kappObj.Id, strings.Join(makeCmd.Env, " "),
			strings.Join(makeCmd.Args, " "))
	} else {
		// run it
		log.Infof("Installing kapp '%s' with command: %s %s",
			kappObj.Id, strings.Join(makeCmd.Env, " "),
			strings.Join(makeCmd.Args, " "))

		err := makeCmd.Run()
		if err != nil {
			return errors.Wrapf(err, "Error installing kapp '%s' with "+
				"command: %s %s. Stderr: %s", kappObj.Id,
				strings.Join(makeCmd.Env, " "), strings.Join(makeCmd.Args, " "),
				stderrBuf.String())
		} else {
			log.Infof("Kapp '%s' successfully %sed", kappObj.Id, makeTarget)
		}
	}

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
