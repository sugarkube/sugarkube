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

type IProvider interface {
	// Returns the name of the provider
	GetName() string
	// Associate provider variables with the provider
	SetVars(map[string]interface{})
	// Returns variables installers should pass on to kapps
	GetInstallerVars() map[string]interface{}
	CustomVarsDirs() []string
	// add a new path to the list of provider vars file paths so we don't need to keep searching
	// the filesystem
	AddVarsPath(path string)
	// get the variable
	VarsFilePaths() []string
}
