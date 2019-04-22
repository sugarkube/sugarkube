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

package plan

import (
	"github.com/sugarkube/sugarkube/internal/pkg/interfaces"
)

// Determines dependencies between kapps in a set of manifests
func findDependencies(manifests []interfaces.IManifest) map[string]nodeDescriptor {
	descriptors := make(map[string]nodeDescriptor, 0)

	var previousInstallable string

	for _, manifest := range manifests {
		for _, installableObj := range manifest.Installables() {
			dependencies := make([]string, 0)

			// if a manifest is marked as being sequential, each kapp depends on the previous one
			if manifest.IsSequential() {
				if previousInstallable != "" {
					dependencies = append(dependencies, previousInstallable)
				}
			} else {
				// otherwise look for explicitly declared dependencies
				// todo - in search for implicit dependencies, e.g. if a kapp uses output from
				//  another kapp, we know there's an implicit dependency between them. The question
				//  is whether that extends to all intermediate kapps - probably yes
				for _, dependency := range installableObj.GetDescriptor().DependsOn {
					dependencies = append(dependencies, dependency)
				}
			}

			descriptors[installableObj.FullyQualifiedId()] = nodeDescriptor{
				dependsOn:      dependencies,
				installableObj: installableObj,
			}

			previousInstallable = installableObj.FullyQualifiedId()
		}
	}

	return descriptors
}
