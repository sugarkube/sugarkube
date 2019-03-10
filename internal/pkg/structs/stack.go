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
package structs

import (
	"github.com/pkg/errors"
	"github.com/sugarkube/sugarkube/internal/pkg/interfaces"
	"github.com/sugarkube/sugarkube/internal/pkg/kapp"
	"github.com/sugarkube/sugarkube/internal/pkg/provider"
	"github.com/sugarkube/sugarkube/internal/pkg/provisioner"
)

// Top-level struct that holds references to instantiations of other objects
// we need to pass around. This is in its own package to avoid circular
// dependencies.
type Stack struct {
	Config      *kapp.StackConfig
	Provider    provider.Provider
	Provisioner provisioner.Provisioner
	Status      *ClusterStatus
}

// Creates a new Stack
func NewStack(config *kapp.StackConfig, provider provider.Provider) (*Stack, error) {

	stack := Stack{
		Config:      config,
		Provider:    provider,
		Provisioner: nil,
		Status: &ClusterStatus{
			isOnline:              false,
			isReady:               false,
			sleepBeforeReadyCheck: 0,
			startedThisRun:        false,
		},
	}

	provisionerImpl, err := provisioner.NewProvisioner(stack.Config.Provisioner, stack)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	stack.Provisioner = provisionerImpl

	return &stack, nil
}

func (s Stack) GetConfig() *kapp.StackConfig {
	return s.Config
}

func (s Stack) GetStatus() interfaces.IClusterStatus {
	return s.Status
}
