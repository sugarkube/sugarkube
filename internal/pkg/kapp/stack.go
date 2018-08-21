package kapp

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/sugarkube/sugarkube/internal/pkg/log"
	"github.com/sugarkube/sugarkube/internal/pkg/vars"
	"gopkg.in/yaml.v2"
	"os"
	"path/filepath"
)

// Hold information about the status of the cluster
type ClusterStatus struct {
	IsOnline              bool   // If true the cluster is online but may not be ready yet
	IsReady               bool   // if true, the cluster is ready to have kapps installed
	StartedThisRun        bool   // if true, the cluster was launched by a provisioner on this invocation
	SleepBeforeReadyCheck uint32 // number of seconds to sleep before polling the cluster for readiness
}

type StackConfig struct {
	Name          string
	FilePath      string
	Provider      string
	Provisioner   string
	Profile       string
	Cluster       string
	VarsFilesDirs []string `yaml:"vars"`
	Manifests     []Manifest
	Status        ClusterStatus
	OnlineTimeout uint32
	ReadyTimeout  uint32
}

// Validates that there aren't multiple manifests in the stack config with the
// same ID, which would break creating caches
func ValidateStackConfig(sc *StackConfig) error {
	ids := map[string]bool{}

	for _, manifest := range sc.Manifests {
		id := manifest.Id

		if _, ok := ids[id]; ok {
			return errors.New(fmt.Sprintf("Multiple manifests exist with "+
				"the same id: %s", id))
		}
	}

	return nil
}

// Loads a stack config from a YAML file and returns it or an error
func LoadStackConfig(name string, path string) (*StackConfig, error) {

	data, err := vars.LoadYamlFile(path)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	stackConfig, ok := data[name]
	if !ok {
		return nil, errors.New(fmt.Sprintf("No stack called '%s' found in stack file %s", name, path))
	}

	log.Debugf("Loaded stack '%s' from file '%s'", name, path)

	// marshal the data we want so we can unmarshal it again into a struct
	stackConfigBytes, err := yaml.Marshal(stackConfig)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	log.Debugf("Stack config bytes:\n%s", stackConfigBytes)

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

	log.Debugf("Loaded stack config: %#v", stack)

	// at this point stack.Manifest only contains a list of URIs. We need to
	// acquire and parse them.
	log.Debug("Parsing manifests")

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

		// this is a mess because the YAML parser requires all fields to be
		// public so we can't enforce a setter :-(
		SetManifestDefaults(&manifest)
		parsedManifest.Id = manifest.Id

		stack.Manifests[i] = *parsedManifest
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
			log.Fatal("Failed to get the path of this binary.")
			panic(err)
		}

		return executable
	}
}
