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

package kapp

import (
	"fmt"
	"github.com/imdario/mergo"
	"github.com/pkg/errors"
	"github.com/sugarkube/sugarkube/internal/pkg/constants"
	"github.com/sugarkube/sugarkube/internal/pkg/convert"
	"github.com/sugarkube/sugarkube/internal/pkg/log"
	"github.com/sugarkube/sugarkube/internal/pkg/utils"
	"github.com/sugarkube/sugarkube/internal/pkg/vars"
	"gopkg.in/yaml.v2"
	"os"
	"path/filepath"
	"strings"
)

// Hold information about the status of the cluster
type ClusterStatus struct {
	IsOnline              bool   // If true the cluster is online but may not be ready yet
	IsReady               bool   // if true, the cluster is ready to have kapps installed
	StartedThisRun        bool   // if true, the cluster was launched by a provisioner on this invocation
	SleepBeforeReadyCheck uint32 // number of seconds to sleep before polling the cluster for readiness
}

type StackConfig struct {
	Name             string
	FilePath         string
	Provider         string
	Provisioner      string
	Account          string
	Region           string
	Profile          string
	Cluster          string
	ProviderVarsDirs []string               `yaml:"providerVarsDirs"`
	providerVars     map[string]interface{} // set after loading and merging all provider vars files
	KappVarsDirs     []string               `yaml:"kappVarsDirs"`
	Manifests        []*Manifest
	TemplateDirs     []string `yaml:"templateDirs"`
	Status           ClusterStatus
	OnlineTimeout    uint32
	ReadyTimeout     uint32
}

// Sets provider vars
func (s *StackConfig) SetProviderVars(vars map[string]interface{}) {
	s.providerVars = vars
}

// Gets provider vars
func (s *StackConfig) GetProviderVars() map[string]interface{} {
	return s.providerVars
}

// Validates that there aren't multiple manifests in the stack config with the
// same ID, which would break creating caches
func ValidateStackConfig(sc *StackConfig) error {
	ids := map[string]bool{}

	for _, manifest := range sc.Manifests {
		id := manifest.Id()

		if _, ok := ids[id]; ok {
			return errors.New(fmt.Sprintf("Multiple manifests exist with "+
				"the same id: %s", id))
		}
	}

	return nil
}

// Loads a stack config from a YAML file and returns it or an error
func LoadStackConfig(name string, path string) (*StackConfig, error) {

	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, errors.Wrap(err, "Can't load non-existent stack file")
	}

	log.Logger.Debugf("Loading stack config from '%s'", path)

	data := map[string]interface{}{}
	err := vars.LoadYamlFile(path, &data)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	stackConfig, ok := data[name]
	if !ok {
		return nil, errors.New(fmt.Sprintf("No stack called '%s' found in stack file %s", name, path))
	}

	log.Logger.Infof("Loaded stack '%s' from file '%s'", name, path)

	// marshal the data we want so we can unmarshal it again into a struct
	stackConfigBytes, err := yaml.Marshal(stackConfig)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	log.Logger.Debugf("Stack config bytes:\n%s", stackConfigBytes)

	stack := StackConfig{
		Name:     name,
		FilePath: path,
		// no-op defaults. Values will be modified by provisioners
		Status: ClusterStatus{
			IsOnline:              false,
			IsReady:               false,
			SleepBeforeReadyCheck: 0,
			StartedThisRun:        false,
		},
	}

	err = yaml.Unmarshal(stackConfigBytes, &stack)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	log.Logger.Debugf("Loaded stack config: %#v", stack)

	// at this point stack.Manifest only contains a list of URIs. We need to acquire and parse them.
	log.Logger.Info("Parsing manifests...")

	for i, manifest := range stack.Manifests {
		// todo - convert these to be managed by acquirers. The file acquirer
		// needs to convert relative paths to absolute.
		uri := manifest.Uri
		if !filepath.IsAbs(uri) {
			uri = filepath.Join(stack.Dir(), uri)
		}

		// parse the manifests and add them back to the stack
		parsedManifest, err := ParseManifestFile(uri)
		if err != nil {
			return nil, errors.WithStack(err)
		}

		// todo - remove this. It should be handled by an acquirer
		//SetManifestDefaults(&manifest)
		parsedManifest.ConfiguredId = manifest.ConfiguredId
		parsedManifest.Overrides = manifest.Overrides

		stack.Manifests[i] = parsedManifest
	}

	return &stack, nil
}

// Returns the directory the stack config was loaded from, or the current
// working directory. This can be used to build relative paths.
func (s *StackConfig) Dir() string {
	if s.FilePath != "" {
		return filepath.Dir(s.FilePath)
	} else {
		executable, err := os.Executable()
		if err != nil {
			log.Logger.Fatal("Failed to get the path of this binary.")
			panic(err)
		}

		return executable
	}
}

// This searches a directory tree from a given root path for files whose values
// should be merged together. If a kapp instance is supplied, additional files
// will be searched for, in addition to stack-specific ones.
func (s *StackConfig) findVarsFiles(kappObj *Kapp) ([]string, error) {
	precedence := []string{
		utils.StripExtension(constants.VALUES_FILE),
		s.Name,
		s.Provider,
		s.Provisioner,
		s.Account,
		s.Region,
		s.Profile,
		s.Cluster,
		constants.PROFILE_DIR,
		constants.CLUSTER_DIR,
	}

	var kappId string

	if kappObj != nil {
		// prepend the kapp ID to the precedence array
		precedence = append([]string{kappObj.Id}, precedence...)

		acquirers, err := kappObj.Acquirers()
		if err != nil {
			return nil, errors.WithStack(err)
		}

		for _, acquirerObj := range acquirers {
			precedence = append(precedence, acquirerObj.Id())

			id, err := acquirerObj.FullyQualifiedId()
			if err != nil {
				return nil, errors.WithStack(err)
			}

			precedence = append(precedence, id)
		}

		kappId = kappObj.Id
	} else {
		kappId = "<no kapp>"
	}

	paths := make([]string, 0)

	for _, searchDir := range s.KappVarsDirs {
		searchPath, err := filepath.Abs(filepath.Join(s.Dir(), searchDir))
		if err != nil {
			return nil, errors.WithStack(err)
		}

		log.Logger.Infof("Searching for files/dirs for kapp '%s' under '%s' with basenames: %s",
			kappId, searchPath, strings.Join(precedence, ", "))

		err = utils.PrecedenceWalk(searchPath, precedence, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return errors.WithStack(err)
			}

			if !info.IsDir() {
				ext := filepath.Ext(path)

				if strings.ToLower(ext) != ".yaml" {
					log.Logger.Debugf("Ignoring non-yaml file: %s", path)
					return nil
				}

				log.Logger.Debugf("Adding kapp var file: %s", path)
				paths = append(paths, path)
			}

			return nil
		})

		if err != nil {
			return nil, errors.WithStack(err)
		}
	}

	log.Logger.Debugf("Kapp var paths for kapp '%s' are: %s", kappId,
		strings.Join(paths, ", "))

	return paths, nil
}

// todo - integrate this. Update findVarsFiles to use the precedence walker too
func (s *StackConfig) findProviderVarsFiles() ([]string, error) {
	precedence := []string{
		utils.StripExtension(constants.VALUES_FILE),
		s.Provider,
		s.Provisioner,
		s.Account,
		s.Profile,
		s.Cluster,
		s.Region,
		"accounts", // todo - pull from the provider
		constants.PROFILE_DIR,
		constants.CLUSTER_DIR,
	}

	paths := make([]string, 0)

	for _, searchDir := range s.ProviderVarsDirs {
		searchPath, err := filepath.Abs(filepath.Join(s.Dir(), searchDir))
		if err != nil {
			return nil, errors.WithStack(err)
		}

		log.Logger.Infof("Searching for files/dirs under '%s' with basenames: %s",
			searchPath, strings.Join(precedence, ", "))

		err = utils.PrecedenceWalk(searchPath, precedence, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return errors.WithStack(err)
			}

			log.Logger.Debugf("Walked to path: %s", path)

			if !info.IsDir() {
				ext := filepath.Ext(path)

				if strings.ToLower(ext) != ".yaml" {
					log.Logger.Debugf("Ignoring non-yaml file: %s", path)
					return nil
				}

				log.Logger.Debugf("Adding var file: %s", path)
				paths = append(paths, path)
			}

			return nil
		})

		if err != nil {
			return nil, errors.WithStack(err)
		}
	}

	log.Logger.Debugf("Provider var paths are: %s", strings.Join(paths, ", "))

	return paths, nil
}

// Merges YAML files that may contain variables. If a a kapp instance is passed
// as a parameter, variables specific to the kapp will also be returned, in addtion
// to those relating to the stack as a whole.
func (s *StackConfig) getVarsFromFiles(kappObj *Kapp) (map[string]interface{}, error) {
	dirs, err := s.findVarsFiles(kappObj)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	values := map[string]interface{}{}

	err = vars.Merge(&values, dirs...)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return values, nil
}

// Returns certain stack data that should be exposed as variables when running kapps
func (s *StackConfig) GetIntrinsicData() map[string]string {
	return map[string]string{
		"name":        s.Name,
		"filePath":    s.FilePath,
		"provider":    s.Provider,
		"provisioner": s.Provisioner,
		"account":     s.Account,
		"region":      s.Region,
		"profile":     s.Profile,
		"cluster":     s.Cluster,
	}
}

// Merges and templates vars from all configured sources. If a kapp instance
// is given, data specific to the kapp will be included in the returned map,
// otherwise only stack-specific variables will be returned.
func (s *StackConfig) TemplatedVars(kappObj *Kapp,
	installerVars map[string]interface{}) (map[string]interface{}, error) {

	stackIntrinsicData := s.GetIntrinsicData()
	// convert the map to the appropriate type and namespace it
	stackConfigMap := map[string]interface{}{
		"stack": convert.MapStringStringToMapStringInterface(stackIntrinsicData),
	}

	mergedVars := map[string]interface{}{}
	err := mergo.Merge(&mergedVars, stackConfigMap, mergo.WithOverride)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	defaultVars := []string{
		s.Provider,
		s.Account, // may be blank depending on the provider
		s.Profile,
		s.Cluster,
		s.Region, // may be blank depending on the provider
	}

	// store additional runtime values under the "sugarkube" key
	installerVars["defaultVars"] = defaultVars
	mergedVars["sugarkube"] = installerVars

	err = mergo.Merge(&mergedVars, s.GetProviderVars(), mergo.WithOverride)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	kappVars, err := s.getVarsFromFiles(kappObj)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	kappIntrinsicDataConverted := map[string]interface{}{}

	if kappObj != nil {
		kappIntrinsicData := kappObj.GetIntrinsicData()
		kappIntrinsicDataConverted = convert.MapStringStringToMapStringInterface(kappIntrinsicData)

		// merge kapp.Vars with the vars from files so kapp.Vars take precedence. Todo - document the order of precedence
		err = mergo.Merge(&kappVars, kappObj.Vars, mergo.WithOverride)
		if err != nil {
			return nil, errors.WithStack(err)
		}
	}

	// namespace kapp variables. This is safer than letting kapp variables overwrite arbitrary values (e.g.
	// so they can't change the target stack, whether the kapp's approved, etc.), but may be too restrictive
	// in certain situations. We'll have to see
	kappIntrinsicDataConverted["vars"] = kappVars

	namespacedKappMap := map[string]interface{}{
		"kapp": kappIntrinsicDataConverted,
	}
	err = mergo.Merge(&mergedVars, namespacedKappMap, mergo.WithOverride)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	templatedVars, err := templateVars(mergedVars)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	yamlData, err := yaml.Marshal(&templatedVars)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	log.Logger.Debugf("Vars after merging and templating:\n%s", yamlData)

	return templatedVars, nil
}
