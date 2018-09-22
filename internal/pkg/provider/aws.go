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
	"fmt"
	"github.com/sugarkube/sugarkube/internal/pkg/kapp"
	"github.com/sugarkube/sugarkube/internal/pkg/log"
	"path/filepath"
)

type AwsProvider struct {
	stackConfigVars Values
	region          string // todo - set this to the dir name when parsing variables
}

const AWS_PROVIDER_NAME = "aws"
const AWS_ACCOUNT_DIR = "accounts"

// Returns a list of directories to load vars files from
func (p *AwsProvider) varsDirs(sc *kapp.StackConfig) ([]string, error) {

	paths := make([]string, 0)

	prefix := sc.Dir()

	for _, path := range sc.VarsFilesDirs {
		// prepend the directory of the stack config file if the path is relative
		if !filepath.IsAbs(path) {
			path = filepath.Join(prefix, path)
			log.Debugf("Prepended dir of stack config to relative path. New path %s", path)
		}

		accountDir := filepath.Join(path, AWS_PROVIDER_NAME, AWS_ACCOUNT_DIR, sc.Account)
		profileDir := filepath.Join(accountDir, PROFILE_DIR, sc.Profile)
		clusterDir := filepath.Join(profileDir, CLUSTER_DIR, sc.Cluster)
		regionDir := filepath.Join(clusterDir, sc.Region)

		p.region = sc.Region

		if err := abortIfNotDir(accountDir,
			fmt.Sprintf("No account directory found at %s", accountDir)); err != nil {
			return nil, err
		}

		if err := abortIfNotDir(profileDir,
			fmt.Sprintf("No profile directory found at %s", profileDir)); err != nil {
			return nil, err
		}

		if err := abortIfNotDir(clusterDir,
			fmt.Sprintf("No cluster directory found at %s", clusterDir)); err != nil {
			return nil, err
		}

		if err := abortIfNotDir(regionDir,
			fmt.Sprintf("No region directory found at %s", regionDir)); err != nil {
			return nil, err
		}

		paths = append(paths, filepath.Join(path))
		paths = append(paths, filepath.Join(path, AWS_PROVIDER_NAME))
		paths = append(paths, filepath.Join(path, AWS_PROVIDER_NAME, AWS_ACCOUNT_DIR))
		paths = append(paths, accountDir)
		paths = append(paths, filepath.Join(path, AWS_PROVIDER_NAME, AWS_ACCOUNT_DIR, sc.Account, PROFILE_DIR))
		paths = append(paths, profileDir)
		paths = append(paths, filepath.Join(path, AWS_PROVIDER_NAME, AWS_ACCOUNT_DIR, sc.Account, PROFILE_DIR, sc.Profile, CLUSTER_DIR))
		paths = append(paths, clusterDir)
		paths = append(paths, regionDir)
	}

	return paths, nil
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
