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
	"fmt"
	"github.com/pkg/errors"
	"github.com/sugarkube/sugarkube/internal/pkg/kapp"
	"github.com/sugarkube/sugarkube/internal/pkg/log"
	"github.com/sugarkube/sugarkube/internal/pkg/provider"
)

type ClusterSot interface {
	isOnline(sc *kapp.StackConfig, providerImpl provider.Provider) (bool, error)
	isReady(sc *kapp.StackConfig, providerImpl provider.Provider) (bool, error)
}

// Implemented ClusterSot names
const KUBECTL = "kubectl"

// Factory that creates ClusterSots
func NewClusterSot(name string) (ClusterSot, error) {
	if name == KUBECTL {
		return KubeCtlClusterSot{}, nil
	}

	return nil, errors.New(fmt.Sprintf("ClusterSot '%s' doesn't exist", name))
}

// Uses an implementation to determine whether the cluster is reachable/online, but it
// may not be ready to install Kapps into yet.
func IsOnline(c ClusterSot, sc *kapp.StackConfig, providerImpl provider.Provider) (bool, error) {
	if sc.Status.IsOnline {
		return true, nil
	}

	online, err := c.isOnline(sc, providerImpl)
	if err != nil {
		return false, errors.WithStack(err)
	}

	if online {
		log.Debug("Cluster is online. Updating cluster status.")
		sc.Status.IsOnline = true
	}

	return online, nil
}

// Uses an implementation to determine whether the cluster is ready to install kapps into
func IsReady(c ClusterSot, sc *kapp.StackConfig, providerImpl provider.Provider) (bool, error) {
	if sc.Status.IsReady {
		return true, nil
	}

	ready, err := c.isReady(sc, providerImpl)
	if err != nil {
		return false, errors.WithStack(err)
	}

	if ready {
		log.Debug("Cluster is ready. Updating cluster status.")
		sc.Status.IsReady = true
	}

	return ready, nil
}
