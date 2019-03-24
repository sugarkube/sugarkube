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

type AwsProvider struct {
	stackConfigVars map[string]interface{}
	region          string
}

const AwsProviderName = "aws"
const AwsAccountDir = "accounts"

// Associate provider variables with the provider
func (p *AwsProvider) SetVars(values map[string]interface{}) {
	p.stackConfigVars = values
}

// Return vars loaded from configs that should be passed on to all kapps by
// installers so kapps can be installed into this provider
func (p *AwsProvider) GetInstallerVars() map[string]interface{} {
	return map[string]interface{}{
		"REGION": p.region,
	}
}

// Returns the name of this provider
func (p *AwsProvider) GetName() string {
	return AwsProviderName
}

// Return static vars dirs names we should search for this provider
func (p *AwsProvider) CustomVarsDirs() []string {
	return []string{
		AwsAccountDir,
		constants.ProfileDir,
		constants.ClusterDir,
	}
}
