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
	"github.com/sugarkube/sugarkube/internal/pkg/interfaces"
	"github.com/sugarkube/sugarkube/internal/pkg/log"
)

type ClusterSot interface {
	isOnline() (bool, error)
	isReady() (bool, error)
	stack() interfaces.IStack
}

// Implemented ClusterSot names
const KUBECTL = "kubectl"

// Factory that creates ClusterSots
func NewClusterSot(name string, iStack interfaces.IStack) (ClusterSot, error) {
	if iStack == nil {
		return nil, errors.New("Stack parameter can't be nil")
	}

	if name == KUBECTL {
		return KubeCtlClusterSot{iStack: iStack}, nil
	}

	return nil, errors.New(fmt.Sprintf("ClusterSot '%s' doesn't exist", name))
}

// Uses an implementation to determine whether the cluster is reachable/online, but it
// may not be ready to install Kapps into yet.
func IsOnline(c ClusterSot) (bool, error) {
	if c.stack().GetStatus().IsOnline() {
		return true, nil
	}

	online, err := c.isOnline()
	if err != nil {
		return false, errors.WithStack(err)
	}

	if online {
		log.Logger.Info("Cluster is online. Updating cluster status.")
		c.stack().GetStatus().SetIsOnline(true)
	}

	return online, nil
}

// Uses an implementation to determine whether the cluster is ready to install kapps into
func IsReady(c ClusterSot) (bool, error) {
	if c.stack().GetStatus().IsReady() {
		return true, nil
	}

	ready, err := c.isReady()
	if err != nil {
		return false, errors.WithStack(err)
	}

	if ready {
		log.Logger.Info("Cluster is ready. Updating cluster status.")
		c.stack().GetStatus().SetIsReady(true)
	}

	return ready, nil
}
