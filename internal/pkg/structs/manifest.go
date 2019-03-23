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
	Source string
	Dest   string
}

// Describes where to find the kapp plus some other data, but isn't the kapp itself
type KappDescriptor struct {
	Id          string
	State       string
	Vars        map[string]interface{}
	PostActions []string `yaml:"post_actions"`
	Sources     []Source
	Templates   []Template
}

type ManifestOptions struct {
	Parallelisation uint16
}

type Manifest struct {
	Options        ManifestOptions
	KappDescriptor []KappDescriptor `yaml:"kapps"`
}
