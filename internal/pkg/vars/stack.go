package vars

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/sugarkube/sugarkube/internal/pkg/log"
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
	Manifests     []string
	Status        ClusterStatus
	OnlineTimeout uint32
	ReadyTimeout  uint32
}

// Loads a stack config from a YAML file and returns it or an error
func LoadStackConfig(name string, path string) (*StackConfig, error) {

	data, err := LoadYamlFile(path)
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
