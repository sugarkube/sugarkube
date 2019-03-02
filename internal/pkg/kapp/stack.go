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
	"github.com/pkg/errors"
	"github.com/sugarkube/sugarkube/internal/pkg/constants"
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
// should be merged together for a kapp based on the values of the stack config
// and the kapp itself.
func (s *StackConfig) findKappVarsFiles(kappObj *Kapp) ([]string, error) {
	validNames := []string{
		s.Name,
		s.Provider,
		s.Provisioner,
		s.Account,
		s.Region,
		s.Profile,
		s.Cluster,
		kappObj.Id,
		constants.PROFILE_DIR,
		constants.CLUSTER_DIR,
	}

	acquirers, err := kappObj.Acquirers()
	if err != nil {
		return nil, errors.WithStack(err)
	}

	for _, acquirerObj := range acquirers {
		validNames = append(validNames, acquirerObj.Id())

		id, err := acquirerObj.FullyQualifiedId()
		if err != nil {
			return nil, errors.WithStack(err)
		}

		validNames = append(validNames, id)
	}

	paths := make([]string, 0)

	for _, searchDir := range s.KappVarsDirs {
		searchPath, err := filepath.Abs(filepath.Join(s.Dir(), searchDir))
		if err != nil {
			return nil, errors.WithStack(err)
		}

		log.Logger.Infof("Searching for files/dirs for kapp '%s' under '%s' with basenames: %s",
			kappObj.FullyQualifiedId(), searchPath, strings.Join(validNames, ", "))

		err = filepath.Walk(searchPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return errors.WithStack(err)
			}

			if info.IsDir() {
				if utils.InStringArray(validNames, info.Name()) || info.Name() == filepath.Base(searchPath) {
					log.Logger.Debugf("Will search kapp var path: %s", path)
					return nil
				} else {
					log.Logger.Debugf("Skipping kapp var dir: %s", path)
					return filepath.SkipDir
				}
			} else {
				basename := filepath.Base(path)
				ext := filepath.Ext(basename)

				if strings.ToLower(ext) != ".yaml" {
					log.Logger.Debugf("Ignoring non-yaml file: %s", path)
					return nil
				}

				nakedBasename := strings.Replace(basename, ext, "", 1)

				if basename == constants.VALUES_FILE || utils.InStringArray(validNames, nakedBasename) {
					log.Logger.Debugf("Adding kapp var file: %s", path)
					// prepend the value to the array to maintain ordering
					paths = append([]string{path}, paths...)
				}
			}

			return nil
		})

		if err != nil {
			return nil, errors.WithStack(err)
		}
	}

	log.Logger.Debugf("Kapp var paths for kapp '%s' are: %s", kappObj.Id,
		strings.Join(paths, ", "))

	return paths, nil
}

// Merges YAML files that may contain values for the given kapp
func (s *StackConfig) GetKappVarsFromFiles(kappObj *Kapp) (map[string]interface{}, error) {
	dirs, err := s.findKappVarsFiles(kappObj)
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
