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
	"github.com/sugarkube/sugarkube/internal/pkg/kapp"
	"github.com/sugarkube/sugarkube/internal/pkg/provider"
	"github.com/sugarkube/sugarkube/internal/pkg/provisioner"
)

type Stack struct {
	Config      *kapp.StackConfig
	provisioner *provisioner.Provisioner
	provider    *provider.Provider
}

// Creates a new Stack object
func NewStack(stackConfig *kapp.StackConfig, provider *provider.Provider) *Stack {
	return &Stack{
		Config:   stackConfig,
		provider: provider,
	}
}
