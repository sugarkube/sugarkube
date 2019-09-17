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

package interfaces

type IProvisioner interface {
	// Returns the ClusterSot for this provisioner
	ClusterSot() IClusterSot
	// Creates a cluster
	Create(dryRun bool) error
	// Deletes a cluster
	Delete(approved bool, dryRun bool) error
	// Returns whether the cluster is already running
	IsAlreadyOnline(dryRun bool) (bool, error)
	// Update the cluster config if supported by the provisioner
	Update(dryRun bool) error
	// We need to use an interface to work with Stack objects to avoid circular dependencies
	GetStack() IStack
	// if the API server is internal we need to set up connectivity to it. Returns a boolean
	// indicating whether connectivity exists (not necessarily if it's been set up, i.e. it
	// might not be necessary to do anything, or it may have already been set up)
	EnsureClusterConnectivity() (bool, error)
	// returns the name of/the path to the underlying binary to use
	Binary() string
	// Shutdown any connectivity to the cluster if any was set up
	Close() error
}
