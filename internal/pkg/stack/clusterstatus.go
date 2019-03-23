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

package stack

// Hold information about the status of the cluster
type ClusterStatus struct {
	isOnline              bool   // If true the cluster is online but may not be ready yet
	isReady               bool   // if true, the cluster is ready to have kapps installed
	startedThisRun        bool   // if true, the cluster was launched by a provisioner on this invocation
	sleepBeforeReadyCheck uint32 // number of seconds to sleep before polling the cluster for readiness
}

func (c *ClusterStatus) IsOnline() bool {
	return c.isOnline
}

func (c *ClusterStatus) SetIsOnline(status bool) {
	c.isOnline = status
}

func (c *ClusterStatus) IsReady() bool {
	return c.isReady
}

func (c *ClusterStatus) SetIsReady(status bool) {
	c.isReady = status
}

func (c *ClusterStatus) StartedThisRun() bool {
	return c.startedThisRun
}

func (c *ClusterStatus) SetStartedThisRun(status bool) {
	c.startedThisRun = status
}

func (c *ClusterStatus) SleepBeforeReadyCheck() uint32 {
	return c.sleepBeforeReadyCheck
}

func (c *ClusterStatus) SetSleepBeforeReadyCheck(time uint32) {
	c.sleepBeforeReadyCheck = time
}
