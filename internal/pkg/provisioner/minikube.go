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

package provisioner

import (
	"bytes"
	"github.com/pkg/errors"
	"github.com/sugarkube/sugarkube/internal/pkg/clustersot"
	"github.com/sugarkube/sugarkube/internal/pkg/kapp"
	"github.com/sugarkube/sugarkube/internal/pkg/log"
	"github.com/sugarkube/sugarkube/internal/pkg/utils"
	"gopkg.in/yaml.v2"
	"os/exec"
)

const MINIKUBE_PROVISIONER_NAME = "minikube"

type MinikubeProvisioner struct {
	clusterSot clustersot.ClusterSot
}

type MinikubeConfig struct {
	Binary string // todo - raise an error if this isn't passed in
	Params struct {
		Global map[string]string
		Start  map[string]string
	}
}

// todo read docs re `minikube profile` to run multiple instances on the same host

// Seconds to sleep after the cluster is online but before checking whether it's ready.
// This gives pods a chance to be launched. If we check immediately there are no pods.
const MINIKUBE_SLEEP_SECONDS_BEFORE_READY_CHECK = 30

func (p MinikubeProvisioner) ClusterSot() (clustersot.ClusterSot, error) {
	if p.clusterSot == nil {
		clusterSot, err := clustersot.NewClusterSot(clustersot.KUBECTL)
		if err != nil {
			return nil, errors.WithStack(err)
		}

		p.clusterSot = clusterSot
	}

	return p.clusterSot, nil
}

// Creates a new minikube cluster
func (p MinikubeProvisioner) create(stackConfig *kapp.StackConfig, dryRun bool) error {

	providerVars := stackConfig.GetProviderVars()

	provisionerValues := providerVars[PROVISIONER_KEY].(map[interface{}]interface{})
	minikubeConfig, err := getMinikubeProvisionerConfig(provisionerValues)
	if err != nil {
		return errors.WithStack(err)
	}
	args := []string{"start"}
	args = parameteriseValues(args, minikubeConfig.Params.Global)
	args = parameteriseValues(args, minikubeConfig.Params.Start)

	var stdoutBuf, stderrBuf bytes.Buffer

	log.Logger.Info("Launching Minikube cluster...")
	err = utils.ExecCommand(minikubeConfig.Binary, args, map[string]string{}, &stdoutBuf,
		&stderrBuf, "", 0, dryRun)
	if err != nil {
		return errors.Wrap(err, "Failed to start a Minikube cluster")
	}

	if !dryRun {
		log.Logger.Infof("Minikube cluster successfully started")
	}

	stackConfig.Status.StartedThisRun = true
	// only sleep before checking the cluster fo readiness if we started it
	stackConfig.Status.SleepBeforeReadyCheck = MINIKUBE_SLEEP_SECONDS_BEFORE_READY_CHECK

	return nil
}

// Returns whether a minikube cluster is already online
func (p MinikubeProvisioner) isAlreadyOnline(stackConfig *kapp.StackConfig) (bool, error) {
	var stdoutBuf, stderrBuf bytes.Buffer

	providerVars := stackConfig.GetProviderVars()

	provisionerValues := providerVars[PROVISIONER_KEY].(map[interface{}]interface{})
	minikubeConfig, err := getMinikubeProvisionerConfig(provisionerValues)
	if err != nil {
		return false, errors.WithStack(err)
	}

	err = utils.ExecCommand(minikubeConfig.Binary, []string{"status"}, map[string]string{},
		&stdoutBuf, &stderrBuf, "", 0, false)
	if err != nil {
		// assume no cluster is up if the command starts but doesn't complete successfully
		if _, ok := errors.Cause(err).(*exec.ExitError); ok {
			return false, nil
		} else {
			// something else, so return an error
			return false, errors.WithStack(err)
		}
	}

	// otherwise assume a cluster is online
	return true, nil
}

// No-op function, required to fully implement the Provisioner interface
func (p MinikubeProvisioner) update(sc *kapp.StackConfig, dryRun bool) error {
	log.Logger.Warn("Updating minikube clusters has no effect. Ignoring.")
	return nil
}

// Parses the provisioner config
func getMinikubeProvisionerConfig(provisionerValues map[interface{}]interface{}) (
	*MinikubeConfig, error) {
	log.Logger.Debugf("Marshalling: %#v", provisionerValues)

	// marshal then unmarshal the provisioner values to get the command parameters
	byteData, err := yaml.Marshal(provisionerValues)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	log.Logger.Debugf("Marshalled to: %s", string(byteData[:]))

	var minikubeConfig MinikubeConfig
	err = yaml.Unmarshal(byteData, &minikubeConfig)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return &minikubeConfig, nil
}
