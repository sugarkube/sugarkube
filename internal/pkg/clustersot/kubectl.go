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
	"github.com/pkg/errors"
	"github.com/sugarkube/sugarkube/internal/pkg/constants"
	"github.com/sugarkube/sugarkube/internal/pkg/interfaces"
	"github.com/sugarkube/sugarkube/internal/pkg/log"
	"github.com/sugarkube/sugarkube/internal/pkg/utils"
	"os/exec"
	"strings"
)

type KubeCtlClusterSot struct {
	iStack interfaces.IStack
}

// todo - make configurable
const kubectlPath = "kubectl"
const kubeContextKey = "kube_context"

const timeoutSeconds = 30

// Tests whether the cluster is online
func (c KubeCtlClusterSot) IsOnline() (bool, error) {
	templatedVars, err := c.iStack.GetTemplatedVars(nil, map[string]interface{}{})
	if err != nil {
		return false, errors.WithStack(err)
	}
	context := templatedVars[kubeContextKey].(string)

	var stdoutBuf, stderrBuf bytes.Buffer

	kubeConfig, _ := c.iStack.GetRegistry().GetString(constants.RegistryKeyKubeConfig)
	envVars := map[string]string{
		"KUBECONFIG": kubeConfig,
	}

	// poll `kubectl --context {{ kube_context }} get namespace`
	err = utils.ExecCommand(kubectlPath, []string{"--context", context, "get", "namespace"},
		envVars, &stdoutBuf, &stderrBuf, "", timeoutSeconds, false)
	if err != nil {
		if _, ok := errors.Cause(err).(*exec.ExitError); ok {
			log.Logger.Info("Cluster isn't online yet - kubectl not getting results")
			return false, nil
		}

		return false, errors.Wrap(err, "Error checking whether cluster is online")
	}

	return true, nil
}

// Tests whether all pods are Ready (or rather whether any pods have a status
// apart from "Running" or "Succeeded")
func (c KubeCtlClusterSot) IsReady() (bool, error) {
	templatedVars, err := c.iStack.GetTemplatedVars(nil, map[string]interface{}{})
	if err != nil {
		return false, errors.WithStack(err)
	}

	context := templatedVars[kubeContextKey].(string)

	var stdoutBuf, stderrBuf bytes.Buffer

	kubeConfig, _ := c.iStack.GetRegistry().GetString(constants.RegistryKeyKubeConfig)

	args := []string{
		"--context", context,
		"-n",
		"kube-system",
		"get", "pod",
		"-o", `go-template={{ range .items }}{{ printf "%s\n" .status.phase }}{{ end }}`,
	}

	envVars := map[string]string{
		"KUBECONFIG": kubeConfig,
	}

	err = utils.ExecCommand(kubectlPath, args, envVars, &stdoutBuf, &stderrBuf,
		"", timeoutSeconds, false)
	if err != nil {
		return false, errors.WithStack(err)
	}

	kubeCtlOutput := stdoutBuf.String()

	log.Logger.Tracef("kubectl readiness check raw output:\n%s", kubeCtlOutput)

	// see whether any statuses apart from "Running" or "Succeeded" were returned
	kubeCtlOutput = strings.Replace(kubeCtlOutput, "Running", "", -1)
	kubeCtlOutput = strings.Replace(kubeCtlOutput, "Succeeded", "", -1)
	kubeCtlOutput = strings.TrimSpace(kubeCtlOutput)

	log.Logger.Tracef("kubectl readiness check after replacements:\n'%s'", kubeCtlOutput)

	return kubeCtlOutput == "", nil
}

func (c KubeCtlClusterSot) Stack() interfaces.IStack {
	return c.iStack
}
