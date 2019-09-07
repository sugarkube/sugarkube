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

// Structs to load a kapp's sugarkube.yaml file

type Template struct {
	Source       string
	Dest         string
	RenderedPath string `yaml:",omitempty"` // path to the rendered template
	Sensitive    bool   // sensitive templates will be templated just-in-time then deleted immediately after
	// executing the kapp. This provides a way of passing secrets to kapps while keeping them off
	// disk as much as possible.
	Conditions []string `yaml:",omitempty"` // the template will only be rendered if all these are true
}

// Outputs generated by a kapp that should be parsed and added to the registry
type Output struct {
	Id   string
	Path string
	// todo - implement - this allows multiple kapps to set the same key. Don't allow kapps to overwrite the outputs under `outputs.kapp` though
	//RegistryPath      string // Path to store the output in the registry (some paths are protected and may not be specified)
	Format    string
	Sensitive bool // sensitive outputs will be deleted after adding the data to the registry to try to prevent
	// secrets lingering on disk
	Conditions []string `yaml:",omitempty"` // outputs will only be loaded if all these are true
}

type Source struct {
	Id      string `yaml:",omitempty"`
	Uri     string
	Options map[string]interface{} `yaml:",omitempty"` // we don't have explicit path/branch fields because this struct must be
	// generic enough for all acquirers, not be specific to git
}

type Action struct {
	Id     string   `yaml:",omitempty"`
	Params []string `yaml:",omitempty"`
}

type RunStep struct {
	Name    string            `yaml:",omitempty"`
	Command string            `yaml:",omitempty"`
	Args    string            `yaml:",omitempty"`
	EnvVars map[string]string `yaml:"env_vars" mapstructure:"env_vars"`
	Stdout  string            `yaml:",omitempty"`                           // path to write stdout to
	Stderr  string            `yaml:",omitempty"`                           // path to write stderr to
	Print   string            `yaml:"print,omitempty" mapstructure:"print"` // if 'verbose', stdout/stderr will be printed to the
	// console when sugarkube is run with the verbose flag. Any other non-empty value will always cause output to be printed regardless
	ExpectedExitCode int      `yaml:"expected_exit_code,omitempty" mapstructure:"expected_exit_code"` // sugarkube will only throw an error if a different exit code is returned
	Conditions       []string `yaml:",omitempty"`
	WorkingDir       string   `yaml:"working_dir,omitempty" mapstructure:"working_dir"`
	MergePriority    *uint8   `yaml:"merge_priority,omitempty" mapstructure:"merge_priority"`
	// pointer so we can tell whether the user has actually set this value or not (otherwise it'd default to the zero value)
	Call string `yaml:",omitempty"`
	// instructs sugarkube to load and parse any outputs defined by the kapp after running
	// this step. Missing outputs won't cause errors though because this can be specified
	// multiple times as different outputs become available.
	LoadOutputs bool `yaml:"load_outputs,omitempty" mapstructure:"load_outputs"`
	Timeout     int  // number of seconds the command must complete within
}

type RunUnit struct {
	WorkingDir   string            `yaml:"working_dir" mapstructure:"working_dir"`
	Conditions   []string          `yaml:",omitempty"`
	EnvVars      map[string]string `yaml:"env_vars" mapstructure:"env_vars"`
	PlanInstall  []RunStep         `yaml:"plan_install,omitempty" mapstructure:"plan_install"`
	ApplyInstall []RunStep         `yaml:"apply_install,omitempty" mapstructure:"apply_install"`
	PlanDelete   []RunStep         `yaml:"plan_delete,omitempty" mapstructure:"plan_delete"`
	ApplyDelete  []RunStep         `yaml:"apply_delete,omitempty" mapstructure:"apply_delete"`
	Output       []RunStep         `yaml:"output,omitempty" mapstructure:"output"`
	Clean        []RunStep         `yaml:"clean,omitempty" mapstructure:"clean"`
}

type Dependency struct {
	Id         string   `yaml:",omitempty"`
	Conditions []string `yaml:",omitempty"`
}

// A struct for an actual sugarkube.yaml file
type KappConfig struct {
	Conditions           []string               `yaml:",omitempty"` // all must be true for the kapp to be processed
	Requires             []string               `yaml:",omitempty"`
	PostInstallActions   []map[string]Action    `yaml:"post_install_actions,omitempty"`
	PostDeleteActions    []map[string]Action    `yaml:"post_delete_actions,omitempty"`
	PreInstallActions    []map[string]Action    `yaml:"pre_install_actions,omitempty"`
	PreDeleteActions     []map[string]Action    `yaml:"pre_delete_actions,omitempty"`
	Templates            map[string]Template    `yaml:",omitempty"`
	Vars                 map[string]interface{} `yaml:",omitempty"`
	RunUnits             map[string]RunUnit     `yaml:"run_units" mapstructure:"run_units"`
	DependsOn            []Dependency           `yaml:"depends_on,omitempty"`   // fully qualified IDs of other kapps this depends on
	IgnoreGlobalDefaults bool                   `yaml:"ignore_global_defaults"` // don't add globally configured defaults for each requirement
	// todo - implement
	//VarsTemplate string		// this will be read as a string, templated then converted to YAML and merged with the Vars map
}

// KappDescriptors describe where to find a kapp plus some other data, but isn't the kapp itself.
// There are two types - one that has certain values declared as lists and one as maps where keys
// are that element's ID. The list version is more concise and is used in manifest files. The
// version with maps is used when overriding values (e.g. in stack files)

type KappDescriptorWithLists struct {
	Id         string `yaml:",omitempty"`
	KappConfig `yaml:",inline"`
	Sources    []Source `yaml:",omitempty"`
	Outputs    []Output `yaml:",omitempty"`
}

type KappDescriptorWithMaps struct {
	Id         string `yaml:",omitempty"`
	KappConfig `yaml:",inline"`
	Sources    map[string]Source `yaml:",omitempty"` // keys are object IDs so values for individual objects can be overridden
	Outputs    map[string]Output `yaml:",omitempty"` // keys are object IDs so values for individual objects can be overridden
}
