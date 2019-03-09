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

package provider

import "github.com/sugarkube/sugarkube/internal/pkg/constants"

type LocalProvider struct {
	stackConfigVars map[string]interface{}
}

const LOCAL_PROVIDER_NAME = "local"

// Associate provider variables with the provider
func (p *LocalProvider) setVars(values map[string]interface{}) {
	p.stackConfigVars = values
}

// Return vars loaded from configs that should be passed on to all kapps by
// installers so kapps can be installed into this provider
func (p *LocalProvider) getInstallerVars() map[string]interface{} {
	return map[string]interface{}{}
}

// Returns the name of this provider
func (p *LocalProvider) getName() string {
	return LOCAL_PROVIDER_NAME
}

// Return static vars dirs names we should search for this provider
func (p *LocalProvider) customVarsDirs() []string {
	return []string{
		constants.PROFILE_DIR,
		constants.CLUSTER_DIR,
	}
}
