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
	"context"
	"github.com/pkg/errors"
	"github.com/sugarkube/sugarkube/internal/pkg/clustersot"
	"github.com/sugarkube/sugarkube/internal/pkg/kapp"
	"github.com/sugarkube/sugarkube/internal/pkg/log"
	"github.com/sugarkube/sugarkube/internal/pkg/provider"
	"github.com/sugarkube/sugarkube/internal/pkg/utils"
	"os/exec"
)

// todo - make configurable
const KOPS_PATH = "kops"

// number of seconds to timeout after while checking whether the Kops cluster
// config exists
const KOPS_COMMAND_TIMEOUT_SECONDS = 10

type KopsProvisioner struct {
	clusterSot clustersot.ClusterSot
}

func (p KopsProvisioner) create(sc *kapp.StackConfig, providerImpl provider.Provider,
	dryRun bool) error {
	log.Debugf("Creating stack with Kops and config: %#v", sc)

	panic("not implemented")
}

func (p KopsProvisioner) ClusterSot() (clustersot.ClusterSot, error) {
	if p.clusterSot == nil {
		clusterSot, err := clustersot.NewClusterSot(clustersot.KUBECTL)
		if err != nil {
			return nil, errors.WithStack(err)
		}

		p.clusterSot = clusterSot
	}

	return p.clusterSot, nil
}

// Returns a bool indicating whether the cluster configuration has already been created
func (p KopsProvisioner) clusterConfigExists(sc *kapp.StackConfig, providerImpl provider.Provider) (bool, error) {

	providerVars := provider.GetVars(providerImpl)
	log.Debugf("Checking if a Kops cluster config exists for values: %#v", providerVars)

	provisionerValues := providerVars[PROVISIONER_KEY].(map[interface{}]interface{})

	args := []string{
		"get",
		"clusters",
		"--state",
		// todo - error checking / defaults here
		provisionerValues["state"].(string),
		provisionerValues["name"].(string),
	}

	var stdoutBuf, stderrBuf bytes.Buffer

	err := utils.ExecWithTimeout(KOPS, args, &stdoutBuf, &stderrBuf,
		KOPS_COMMAND_TIMEOUT_SECONDS, false)

	if err != nil {
		if err == context.DeadlineExceeded {
			return false, errors.Wrap(err,
				"Timed out trying to retrieve kops cluster config. "+
					"Check your credentials.")
		}

		if _, ok := err.(*exec.ExitError); ok {
			log.Debug("Cluster config doesn'te exist")
			return false, nil
		} else {
			return false, errors.Wrap(err, "Error fetching kops clusters")
		}
	}

	return true, nil
}

// Returns a boolean indicating whether the cluster is already online
func (p KopsProvisioner) isAlreadyOnline(sc *kapp.StackConfig, providerImpl provider.Provider) (bool, error) {
	configExists, err := p.clusterConfigExists(sc, providerImpl)
	if err != nil {
		return false, errors.WithStack(err)
	}

	if !configExists {
		return false, nil
	}

	clusterSot, err := p.ClusterSot()
	if err != nil {
		return false, errors.WithStack(err)
	}

	online, err := clustersot.IsOnline(clusterSot, sc, providerImpl)
	if err != nil {
		return false, errors.WithStack(err)
	}

	return online, nil
}

// No-op function, required to fully implement the Provisioner interface
func (p KopsProvisioner) update(sc *kapp.StackConfig, providerImpl provider.Provider) error {
	panic("not implemented")
}
