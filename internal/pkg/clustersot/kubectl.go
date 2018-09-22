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

package clustersot

import (
	"bytes"
	"fmt"
	"github.com/pkg/errors"
	"github.com/sugarkube/sugarkube/internal/pkg/kapp"
	"github.com/sugarkube/sugarkube/internal/pkg/log"
	"github.com/sugarkube/sugarkube/internal/pkg/provider"
	"github.com/sugarkube/sugarkube/internal/pkg/utils"
	"os"
	"os/exec"
)

type KubeCtlClusterSot struct {
	ClusterSot
}

// todo - make configurable
const KUBECTL_PATH = "kubectl"
const KUBE_CONTEXT_KEY = "kube_context"

// Tests whether the cluster is online
func (c KubeCtlClusterSot) isOnline(sc *kapp.StackConfig, providerImpl provider.Provider) (bool, error) {
	providerVars := provider.GetVars(providerImpl)
	context := providerVars[KUBE_CONTEXT_KEY].(string)

	var stdoutBuf, stderrBuf bytes.Buffer

	// poll `kubectl --context {{ kube_context }} get namespace`
	err := utils.ExecCommand(KUBECTL_PATH, []string{"--context", context, "get", "namespace"},
		&stdoutBuf, &stderrBuf, "", 30, false)
	if err != nil {
		if _, ok := errors.Cause(err).(*exec.ExitError); ok {
			log.Debug("Cluster isn't online yet - kubectl not getting results")
			return false, nil
		}

		return false, errors.Wrap(err, "Error checking whether cluster is online")
	}

	return true, nil
}

// Tests whether all pods are Ready
func (c KubeCtlClusterSot) isReady(sc *kapp.StackConfig, providerImpl provider.Provider) (bool, error) {
	providerVars := provider.GetVars(providerImpl)
	context := providerVars[KUBE_CONTEXT_KEY].(string)

	// todo - simplify this by using ExecCommand to get the data from kubectl with a timeout,
	// then just feed that to grep on its stdin instead of piping directly.
	userEnv := os.Environ()
	var kubeCtlStderr, grepStdout bytes.Buffer

	kubeCtlCmd := exec.Command(KUBECTL_PATH, "--context", context, "-n", "kube-system",
		"get", "pod", "-o", "go-template=\"{{ range .items }}{{ printf \"%%s\\n\" .status.phase }}{{ end }}\"")
	kubeCtlCmd.Env = userEnv
	kubeCtlCmd.Stderr = &kubeCtlStderr
	kubeCtlStdout, err := kubeCtlCmd.StdoutPipe()
	if err != nil {
		return false, errors.WithStack(err)
	}

	grepCmd := exec.Command("grep", "-v", "Running")
	grepCmd.Env = userEnv
	grepCmd.Stdin = kubeCtlStdout
	grepCmd.Stdout = &grepStdout

	err = grepCmd.Start()
	if err != nil {
		return false, errors.Wrap(err, "Failed to run grep")
	}

	err = kubeCtlCmd.Start()
	if err != nil {
		return false, errors.Wrap(err, "Failed to run kubectl")
	}

	err = kubeCtlCmd.Wait()
	if err != nil {
		if _, ok := err.(*exec.ExitError); ok {
			if kubeCtlStderr.String() != "" {
				errMsg := fmt.Sprintf("kubectl exited with %s", kubeCtlStderr.String())
				log.Fatalf(errMsg)
				return false, errors.Wrap(err, errMsg)
			} else {
				return false, nil
			}
		}

		return false, errors.Wrap(err, "kubectl terminated badly")
	}

	err = grepCmd.Wait()
	if err != nil {
		return false, errors.Wrap(err, "grep terminated badly")
	}

	// some funkiness probably with new lines means that even if grep return
	// no output, the length of its stdout buffer isn't 0, but this is
	// good enough...
	return grepStdout.Len() < 5, nil
}
