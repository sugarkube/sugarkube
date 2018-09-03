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

import (
	"github.com/sugarkube/sugarkube/internal/pkg/kapp"
)

type AwsProvider struct {
	stackConfigVars Values
	region          string // todo - set this to the dir name when parsing variables
}

func (p *AwsProvider) varsDirs(stackConfig *kapp.StackConfig) ([]string, error) {
	return []string{
		"/cat",
		"/dog",
	}, nil
}

// Associate provider variables with the provider
func (p *AwsProvider) setVars(values Values) {
	p.stackConfigVars = values
}

// Returns the variables loaded by the Provider
func (p *AwsProvider) getVars() Values {
	return p.stackConfigVars
}

// Return vars loaded from configs that should be passed on to all kapps by
// installers so kapps can be installed into this provider
func (p *AwsProvider) getInstallerVars() Values {
	return Values{
		"REGION": p.region,
	}
}
