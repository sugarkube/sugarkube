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

import (
	"github.com/sugarkube/sugarkube/internal/pkg/acquirer"
	"github.com/sugarkube/sugarkube/internal/pkg/structs"
)

// this encapsulates different package formats that sugarkube can install in
// a target stack
type IInstallable interface {
	Id() string
	FullyQualifiedId() string
	ManifestId() string
	PreInstallActions() []structs.Action
	PreDeleteActions() []structs.Action
	PostInstallActions() []structs.Action
	PostDeleteActions() []structs.Action
	GetDescriptor() structs.KappDescriptorWithMaps
	LoadConfigFile(workspaceDir string) error
	SetWorkspaceDir(workspaceDir string) error
	GetCacheDir() string
	GetConfigFileDir() string
	Acquirers() (map[string]acquirer.Acquirer, error)
	TemplateDescriptor(templateVars map[string]interface{}) error
	Vars(stack IStack) (map[string]interface{}, error)
	AddDescriptor(config structs.KappDescriptorWithMaps, prepend bool) error
	RenderTemplates(templateVars map[string]interface{}, stackConfig IStackConfig,
		dryRun bool) error
	GetOutputs(ignoreMissing bool, dryRun bool) (map[string]interface{}, error)
	HasOutputs() bool
	GetLocalRegistry() IRegistry
	SetLocalRegistry(registry IRegistry)
}
