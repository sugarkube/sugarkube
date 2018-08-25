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
			"not implemented yet: ", strings.Join(makefilePaths, ", ")))
	}

	makefilePath := makefilePaths[0]

	// search for values-<env>.yaml files where env could also be the cluster/
	// profile/etc. Todo - think where to get the pattern `values-<var>.yaml`
	// from that doesn't make us rely on Helm. Answer: paramertisers

	// create the env vars
	envVars := []string{
		fmt.Sprintf("APPROVED=%v", approved),
		"CLUSTER=" + stackConfig.Cluster,
		"PROFILE=" + stackConfig.Profile,
		"PROVIDER=" + stackConfig.Provider,
		// Helm-specific env var names - todo use something that understand helm to add these in
		"NAMESPACE=" + kappObj.Id, // todo - permit overrides
		"RELEASE=" + kappObj.Id,
		//"CHART_DIR=???"			// todo - is this necessary?
	}

	// todo - there should be something that understand Helm and that takes stuff
	// from the provider to return the vars that the installer cares about for Helm
	// kapps. I.e. we shouldn't return the KUBE_CONTEXT for non-k8s/helm kapps.
	for k, v := range provider.GetInstallerVars(i.provider) {
		upperKey := strings.ToUpper(k)
		kv := fmt.Sprintf("%s=%s", upperKey, v)
		envVars = append(envVars, kv)
	}

	// build the command
	var stderrBuf bytes.Buffer

	// make command
	makeCmd := exec.Command(makefilePath, TARGET_INSTALL)
	makeCmd.Dir = filepath.Dir(makefilePath)
	makeCmd.Env = envVars
	makeCmd.Stderr = &stderrBuf

	if dryRun {
		log.Infof("Dry run. Would install kapp '%s' with command: %s %s",
			kappObj.Id, strings.Join(makeCmd.Env, " "),
			strings.Join(makeCmd.Args, " "))
	} else {
		// run it
		err := makeCmd.Run()
		if err != nil {
			return errors.Wrapf(err, "Error installing kapp '%s' with "+
				"command: %s %s. Stderr: %s", kappObj.Id,
				strings.Join(makeCmd.Env, " "), strings.Join(makeCmd.Args, " "),
				stderrBuf.String())
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
