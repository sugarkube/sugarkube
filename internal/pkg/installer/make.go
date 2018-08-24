package installer

import (
	"bytes"
	"github.com/pkg/errors"
	"github.com/sugarkube/sugarkube/internal/pkg/kapp"
	"os/exec"
	"path/filepath"
)

// Installs kapps with make
type MakeInstaller struct{}

const TARGET_INSTALL = "install"
const TARGET_DESTROY = "destroy"

func (i MakeInstaller) install(kappObj *kapp.Kapp, stackConfig *kapp.StackConfig) error {
	// search for the Makefile
	makefilePath := "..."

	// search for values-<env>.yaml files where env could also be the cluster/
	// profile/etc. Todo - think where to get the pattern `values-<var>.yaml`
	// from that doesn't make us rely on Helm.

	// create the env vars
	envVars := []string{
		"APPROVED=false", // todo - parameterise
		"CLUSTER=" + stackConfig.Cluster,
		"PROFILE=" + stackConfig.Profile,
		"PROVIDER=" + stackConfig.Provider,
		// Provider-specific
		//"REGION=" + stackConfig.Provider.Region,
		//// Helm-specific todo
		// comes from stackConfigVars
		//"KUBE_CONTEXT=" + stackConfig.Provider.Context(),
		//"NAMESPACE=" + ??,
		//"RELEASE=????,
		//"CHART_DIR=???"

	}

	// build the command
	var stderrBuf bytes.Buffer

	// make command
	makeCmd := exec.Command(makefilePath, TARGET_INSTALL)
	makeCmd.Dir = filepath.Dir(makefilePath)
	makeCmd.Args = envVars
	makeCmd.Stderr = &stderrBuf

	// run it
	err := makeCmd.Run()
	if err != nil {
		return errors.Wrapf(err, "Error installing kapp '%s' with "+
			"command: %#v. Stderr: %s", kappObj.Id, makeCmd, stderrBuf.String())
	}

	return nil
}

func (i MakeInstaller) destroy(kappObj *kapp.Kapp, stackConfig *kapp.StackConfig) error {
	return nil
}
