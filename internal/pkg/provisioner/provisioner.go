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
	"github.com/sugarkube/sugarkube/internal/pkg/kapp"
	"github.com/sugarkube/sugarkube/internal/pkg/log"
	"time"
)

type Provisioner interface {
	// Returns the ClusterSot for this provisioner
	ClusterSot() (clustersot.ClusterSot, error)
	// Creates a cluster
	create(sc *kapp.StackConfig, dryRun bool) error
	// Returns whether the cluster is already running
	isAlreadyOnline(sc *kapp.StackConfig) (bool, error)
	// Update the cluster config if supported by the provisioner
	update(sc *kapp.StackConfig, dryRun bool) error
}

// key in Values that relates to this provisioner
const PROVISIONER_KEY = "provisioner"

// Factory that creates providers
func NewProvisioner(name string, stackConfig *kapp.StackConfig) (Provisioner, error) {
	if name == MINIKUBE_PROVISIONER_NAME {
		return MinikubeProvisioner{}, nil
	}

	if name == KOPS_PROVISIONER_NAME {
		return newKopsProvisioner(stackConfig), nil
	}

	if name == NOOP_PROVISIONER_NAME {
		return NoOpProvisioner{}, nil
	}

	return nil, errors.New(fmt.Sprintf("Provisioner '%s' doesn't exist", name))
}

// Creates a cluster using an implementation of a Provisioner
func Create(p Provisioner, sc *kapp.StackConfig, dryRun bool) error {
	return p.create(sc, dryRun)
}

// Updates a cluster using an implementation of a Provisioner
func Update(p Provisioner, sc *kapp.StackConfig, dryRun bool) error {
	return p.update(sc, dryRun)
}

// Return whether the cluster is already online
func IsAlreadyOnline(p Provisioner, stackConfig *kapp.StackConfig) (bool, error) {

	log.Logger.Infof("Checking whether cluster '%s' is already online...", stackConfig.Cluster)

	online, err := p.isAlreadyOnline(stackConfig)
	if err != nil {
		return false, errors.WithStack(err)
	}

	if online {
		log.Logger.Infof("Cluster '%s' is online", stackConfig.Cluster)
	} else {
		log.Logger.Infof("Cluster '%s' is not online", stackConfig.Cluster)
	}

	stackConfig.Status.IsOnline = online
	return online, nil
}

// Wait for a cluster to come online, then to become ready.
func WaitForClusterReadiness(p Provisioner, sc *kapp.StackConfig) error {
	clusterSot, err := p.ClusterSot()
	if err != nil {
		return errors.WithStack(err)
	}

	log.Logger.Infof("Checking whether the cluster is online... Will try for %d seconds",
		sc.OnlineTimeout)

	clusterWasOffline := false
	offlineInfoMessageShown := false

	timeoutTime := time.Now().Add(time.Second * time.Duration(sc.OnlineTimeout))
	for time.Now().Before(timeoutTime) {
		online, err := clustersot.IsOnline(clusterSot, sc)
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
					"for %d seconds...", sc.OnlineTimeout)
				offlineInfoMessageShown = true
			}

			log.Logger.Debug("Cluster isn't online. Sleeping...")
			time.Sleep(time.Duration(5) * time.Second)
		}
	}

	if !sc.Status.IsOnline {
		return errors.New("Timed out waiting for the cluster to come online")
	}

	// only sleep before checking readiness if the cluster was initially offline
	sleepTime := sc.Status.SleepBeforeReadyCheck
	if clusterWasOffline || sc.Status.StartedThisRun && sleepTime > 0 {
		log.Logger.Infof("Sleeping for %d seconds before checking cluster readiness...", sleepTime)
		time.Sleep(time.Second * time.Duration(sleepTime))
	}

	log.Logger.Infof("Checking whether the cluster is ready...")

	readinessTimeoutTime := time.Now().Add(time.Second * time.Duration(sc.OnlineTimeout))
	for time.Now().Before(readinessTimeoutTime) {
		ready, err := clustersot.IsReady(clusterSot, sc)
		if err != nil {
			return errors.WithStack(err)
		}

		if ready {
			log.Logger.Info("Cluster is ready")
			break
		} else {
			log.Logger.Info("Cluster isn't ready. Sleeping...")
			time.Sleep(time.Duration(5) * time.Second)
		}
	}

	if !sc.Status.IsReady {
		return errors.New("Timed out waiting for the cluster to become ready")
	}

	return nil
}
