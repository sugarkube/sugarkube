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
	"github.com/pkg/errors"
	"github.com/sugarkube/sugarkube/internal/pkg/constants"
	"github.com/sugarkube/sugarkube/internal/pkg/interfaces"
	"github.com/sugarkube/sugarkube/internal/pkg/log"
	"github.com/sugarkube/sugarkube/internal/pkg/structs"
	"strings"
)

// Determines dependencies between kapps in a set of manifests
func findDependencies(manifests []interfaces.IManifest) (map[string]nodeDescriptor, error) {
	descriptors := make(map[string]nodeDescriptor, 0)

	var previousInstallable string

	for _, manifest := range manifests {
		previousInstallable = ""
		for _, installableObj := range manifest.Installables() {
			dependencies := make([]structs.Dependency, 0)

			log.Logger.Tracef("Candidate dependency: %#v", installableObj)

			// if a manifest is marked as being sequential, each kapp depends on the previous one
			if manifest.IsSequential() {
				log.Logger.Debugf("Manifest '%s' is sequential", manifest.Id())
				if previousInstallable != "" {
					log.Logger.Tracef("Adding previous installable '%s' as a dependency",
						previousInstallable)
					dependencies = append(dependencies, structs.Dependency{Id: previousInstallable})
				} else if len(installableObj.GetDescriptor().DependsOn) > 0 {
					log.Logger.Tracef("Installable '%s' depends on %v", installableObj.FullyQualifiedId(),
						installableObj.GetDescriptor().DependsOn)
					for _, dependency := range installableObj.GetDescriptor().DependsOn {
						// fully-qualify the dependency if it's not already
						if !strings.Contains(dependency.Id, constants.NamespaceSeparator) {
							dependency.Id = strings.Join([]string{installableObj.ManifestId(), dependency.Id},
								constants.NamespaceSeparator)
						}
						dependencies = append(dependencies, dependency)
					}
				}
			} else {
				// otherwise look for explicitly declared dependencies
				// todo - in search for implicit dependencies, e.g. if a kapp uses output from
				//  another kapp, we know there's an implicit dependency between them. The question
				//  is whether that extends to all intermediate kapps - probably yes
				log.Logger.Tracef("Installable '%s' depends on %v", installableObj.FullyQualifiedId(),
					installableObj.GetDescriptor().DependsOn)
				for _, dependency := range installableObj.GetDescriptor().DependsOn {
					// fully-qualify the dependency if it's not already
					if !strings.Contains(dependency.Id, constants.NamespaceSeparator) {
						dependency.Id = strings.Join([]string{installableObj.ManifestId(), dependency.Id},
							constants.NamespaceSeparator)
					}
					dependencies = append(dependencies, dependency)
				}
			}

			// push a new descriptor into the kapp to update its list of dependencies
			configDescriptor := structs.KappDescriptorWithMaps{
				KappConfig: structs.KappConfig{
					DependsOn: dependencies,
				},
			}
			err := installableObj.AddDescriptor(configDescriptor, false)
			if err != nil {
				return nil, errors.WithStack(err)
			}

			descriptors[installableObj.FullyQualifiedId()] = nodeDescriptor{
				installableObj: installableObj,
			}

			previousInstallable = installableObj.FullyQualifiedId()
		}
	}

	return descriptors, nil
}
