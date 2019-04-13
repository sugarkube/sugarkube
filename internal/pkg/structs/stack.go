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

// Structs to load a stack YAML file

// Describes where to find the manifest plus some other data, but isn't the manifest itself
type ManifestDescriptor struct {
	Id  string // a default will be used if not explicitly set. Used to namespace cache entries
	Uri string

	// todo - we should get rid of the Id and Uri fields and just use a Source and acquirers:
	//Source `yaml:",inline"`

	Overrides map[string]KappDescriptorWithMaps // the map key is the kappDescriptor ID
}

// todo - allow defaults to be specified to be used as overrides/defaults for all manifests in a stack
//  also create manifestGroups and allow each group to be executed separately
type StackFile struct {
	Name     string // this is in the YAML file, but is the key that the config is under
	FilePath string // this is immutable too and is intrinsically related to the config so although it's
	// not directly in the YAML, an exception has been made
	Provider            string
	Provisioner         string
	Account             string
	Region              string
	Profile             string
	Cluster             string
	ProviderVarsDirs    []string             `yaml:"providerVarsDirs"`
	KappVarsDirs        []string             `yaml:"kappVarsDirs"`
	ManifestDescriptors []ManifestDescriptor `yaml:"manifests"` // this struct should be immutable, so don't store pointers
	TemplateDirs        []string             `yaml:"templateDirs"`
}
