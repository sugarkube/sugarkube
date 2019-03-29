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

// Structs to load a manifest YAML file

type Template struct {
	Source    string
	Dest      string
	Sensitive bool // sensitive templates will be templated just-in-time then deleted immediately after
	// executing the kapp. This provides a way of passing secrets to kapps while keeping them off
	// disk as much as possible.
}

// Describes where to find the kapp plus some other data, but isn't the kapp itself
type KappDescriptor struct {
	Id         string
	State      string
	FilePath   string // path of the file the descriptor was loaded from (for resolving relative paths)
	KappConfig `yaml:",inline"`
}

type ManifestOptions struct {
	Parallelisation uint16
}

type Manifest struct {
	FilePath       string     // path of the manifest file (for resolving relative paths)
	Defaults       KappConfig // Defaults that apply to all kapps in the manifest
	States         []string   // Basenames of files to merge in with the highest priority
	Options        ManifestOptions
	KappDescriptor []KappDescriptor `yaml:"kapps"`
}
