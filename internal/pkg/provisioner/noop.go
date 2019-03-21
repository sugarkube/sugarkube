/*
 * Copyright 2019 The Sugarkube Authors
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
	"github.com/sugarkube/sugarkube/internal/pkg/clustersot"
	"github.com/sugarkube/sugarkube/internal/pkg/interfaces"
	"github.com/sugarkube/sugarkube/internal/pkg/log"
)

const NoopProvisionerName = "none"

// A no-op provisioner that doesn't create a cluster at all. This can be useful
// if you just want to create raw resources without a K8s cluster, e.g. to
// create a transit VPC, launch some EC2s with CloudFormation, etc.
type NoOpProvisioner struct {
	stack interfaces.IStack
}

type NoopConfig struct {
}

func (p NoOpProvisioner) ClusterSot() clustersot.ClusterSot {
	return nil
}

func (p NoOpProvisioner) iStack() interfaces.IStack {
	return p.stack
}

// Creates a new noop cluster
func (p NoOpProvisioner) create(dryRun bool) error {

	log.Logger.Infof("Noop provisioner - no cluster will be created")
	return nil
}

// Returns whether a noop cluster is already online
func (p NoOpProvisioner) isAlreadyOnline(dryRun bool) (bool, error) {

	log.Logger.Infof("Noop provisioner - pretending a cluster is online")
	// return that the cluster is online
	return true, nil
}

// No-op function, required to fully implement the Provisioner interface
func (p NoOpProvisioner) update(dryRun bool) error {

	log.Logger.Infof("Noop provisioner - no cluster will be updated")
	return nil
}

// No special connectivity is required for this provisioner
func (p NoOpProvisioner) ensureClusterConnectivity() (bool, error) {
	return true, nil
}
