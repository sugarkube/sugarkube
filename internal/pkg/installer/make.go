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

func (i MakeInstaller) install(kappObj *kapp.Kapp, stackConfig *kapp.StackConfig,
	dryRun bool) error {
	// search for the Makefile
	makefilePath := "/some/path/Makefile"

	// search for values-<env>.yaml files where env could also be the cluster/
	// profile/etc. Todo - think where to get the pattern `values-<var>.yaml`
	// from that doesn't make us rely on Helm.

	// create the env vars
	envVars := []string{
		"APPROVED=false", // todo - parameterise
		"CLUSTER=" + stackConfig.Cluster,
		"PROFILE=" + stackConfig.Profile,
		"PROVIDER=" + stackConfig.Provider,
		//// Helm-specific todo
		//"NAMESPACE=" + ??,
		//"RELEASE=????,
		//"CHART_DIR=???"
	}

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

func (i MakeInstaller) destroy(kappObj *kapp.Kapp, stackConfig *kapp.StackConfig, dryRun bool) error {
	return nil
}
