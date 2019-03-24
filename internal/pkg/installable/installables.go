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

package installable

import (
	"github.com/sugarkube/sugarkube/internal/pkg/acquirer"
	"github.com/sugarkube/sugarkube/internal/pkg/structs"
)

// These are defined here to avoid circular dependencies
type iStackConfig interface {
	Name() string
	Provider() string
	Provisioner() string
	Account() string
	Region() string
	Profile() string
	Cluster() string
	KappVarsDirs() []string
	Dir() string
	TemplateDirs() []string
}

type iStack interface {
	GetConfig() iStackConfig
}

// this encapsulates different package formats that sugarkube can install in
// a target stack
type Installable interface {
	Id() string
	FullyQualifiedId() string
	ManifestId() string
	State() string
	PostActions() []string
	SetRootCacheDir(cacheDir string)
	Acquirers() ([]acquirer.Acquirer, error)
	RefreshConfig(templateVars map[string]interface{}) error
	Vars(stack iStack) (map[string]interface{}, error)
	RenderTemplates(templateVars map[string]interface{}, stackConfig iStackConfig,
		dryRun bool) ([]string, error)
}

func New(manifestId string, descriptor structs.KappDescriptor) (Installable, error) {
	return &Kapp{
		manifestId: manifestId,
		descriptor: descriptor,
	}, nil
}
