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
	"fmt"
	"github.com/pkg/errors"
	"github.com/sugarkube/sugarkube/internal/pkg/clustersot"
	"github.com/sugarkube/sugarkube/internal/pkg/interfaces"
	"github.com/sugarkube/sugarkube/internal/pkg/log"
	"time"
)

const shortSleepTime = 5

type Provisioner interface {
	// Returns the ClusterSot for this provisioner
	ClusterSot() clustersot.ClusterSot
	// Creates a cluster
	create(dryRun bool) error
	// Returns whether the cluster is already running
	isAlreadyOnline(dryRun bool) (bool, error)
	// Update the cluster config if supported by the provisioner
	update(dryRun bool) error
	// We need to use an interface to work with Stack objects to avoid circular dependencies
	iStack() interfaces.IStack
	// if the API server is internal we need to set up connectivity to it. Returns a boolean
	// indicating whether connectivity exists (not necessarily if it's been set up, i.e. it
	// might not be necessary to do anything, or it may have already been set up)
	ensureClusterConnectivity() (bool, error)
}

// key in Values that relates to this provisioner
const ProvisionerKey = "provisioner"

// Factory that creates providers
func New(name string, stack interfaces.IStack,
	clusterSot clustersot.ClusterSot) (Provisioner, error) {
	if stack == nil {
		return nil, errors.New("Stack parameter can't be nil")
	}

	if name == MinikubeProvisionerName {
		minikubeProvisioner, err := newMinikubeProvisioner(stack, clusterSot)
		if err != nil {
			return nil, errors.WithStack(err)
		}

		return *minikubeProvisioner, nil
	}

	if name == kopsProvisionerName {
		kopsProvisioner, err := newKopsProvisioner(stack, clusterSot)
		if err != nil {
			return nil, errors.WithStack(err)
		}

		return kopsProvisioner, nil
	}

	if name == NoopProvisionerName {
		return NoOpProvisioner{
			stack: stack,
		}, nil
	}

	return nil, errors.New(fmt.Sprintf("Provisioner '%s' doesn't exist", name))
}

// Creates a cluster using an implementation of a Provisioner
func Create(p Provisioner, dryRun bool) error {
	return p.create(dryRun)
}

// Updates a cluster using an implementation of a Provisioner
func Update(p Provisioner, dryRun bool) error {
	return p.update(dryRun)
}

// Return whether the cluster is already online
func IsAlreadyOnline(p Provisioner, dryRun bool) (bool, error) {

	clusterName := p.iStack().GetConfig().Name

	log.Logger.Infof("Checking whether cluster '%s' is already online...",
		clusterName)

	connected, err := p.ensureClusterConnectivity()
	if err != nil {
		return false, errors.WithStack(err)
	}

	if !connected {
		log.Logger.Infof("Couldn't establish a connection to cluster '%s'", clusterName)
		return false, nil
	}

	online, err := p.isAlreadyOnline(dryRun)
	if err != nil {
		return false, errors.WithStack(err)
	}

	if online {
		log.Logger.Infof("Cluster '%s' is online", clusterName)
	} else {
		log.Logger.Infof("Cluster '%s' is not online", clusterName)
	}

	p.iStack().GetStatus().SetIsOnline(online)
	return online, nil
}

// Wait for a cluster to come online, then to become ready.
func WaitForClusterReadiness(p Provisioner) error {
	clusterSot := p.ClusterSot()

	onlineTimeout := p.iStack().GetConfig().OnlineTimeout

	log.Logger.Infof("Checking whether the cluster is online... Will "+
		"try for %d seconds", onlineTimeout)

	clusterWasOffline := false
	offlineInfoMessageShown := false

	timeoutTime := time.Now().Add(time.Second * time.Duration(onlineTimeout))
	for time.Now().Before(timeoutTime) {
		connected, err := p.ensureClusterConnectivity()
		if err != nil {
			return errors.WithStack(err)
		}

		if !connected {
			log.Logger.Infof("Couldn't establish a connection to the " +
				"cluster. Sleeping before retrying...")
			time.Sleep(shortSleepTime * time.Second)
			continue
		}

		online, err := clustersot.IsOnline(clusterSot)
		if err != nil {
			return errors.WithStack(err)
		}

		if online {
			log.Logger.Info("Cluster is online")
			break
		} else {
			clusterWasOffline = true

			// only show this info message once to avoid noisy logs
			if !offlineInfoMessageShown {
				log.Logger.Infof("Cluster isn't online. Will keep retrying "+
					"for %d seconds...", onlineTimeout)
				offlineInfoMessageShown = true
			}

			log.Logger.Debug("Cluster isn't online. Sleeping...")
			time.Sleep(shortSleepTime * time.Second)
		}
	}

	if !p.iStack().GetStatus().IsOnline() {
		return errors.New("Timed out waiting for the cluster to come online")
	}

	// only sleep before checking readiness if the cluster was initially offline
	sleepTime := p.iStack().GetStatus().SleepBeforeReadyCheck()
	if clusterWasOffline || p.iStack().GetStatus().StartedThisRun() && sleepTime > 0 {
		log.Logger.Infof("Sleeping for %d seconds before checking cluster readiness...", sleepTime)
		time.Sleep(time.Second * time.Duration(sleepTime))
	}

	log.Logger.Infof("Checking whether the cluster is ready...")

	readinessTimeoutTime := time.Now().Add(time.Second * time.Duration(onlineTimeout))
	for time.Now().Before(readinessTimeoutTime) {
		ready, err := clustersot.IsReady(clusterSot)
		if err != nil {
			return errors.WithStack(err)
		}

		if ready {
			log.Logger.Info("Cluster is ready")
			break
		} else {
			log.Logger.Info("Cluster isn't ready. Sleeping...")
			time.Sleep(shortSleepTime * time.Second)
		}
	}

	if !p.iStack().GetStatus().IsReady() {
		return errors.New("Timed out waiting for the cluster to become ready")
	}

	return nil
}
