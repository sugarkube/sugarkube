package vars

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/sugarkube/sugarkube/internal/pkg/log"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
	"path/filepath"
)

// Hold information about the status of the cluster
type ClusterStatus struct {
	IsOnline              bool  // If true the cluster is online but may not be ready yet
	IsReady               bool  // if true, the cluster is ready to have kapps installed
	StartedThisRun        bool  // if true, the cluster was launched by a provisioner on this invocation
	SleepBeforeReadyCheck uint8 // number of seconds to sleep before polling the cluster for readiness
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
	OnlineTimeout uint8
	ReadyTimeout  uint8
}

// Loads a stack config from a YAML file and returns it or an error
func LoadStackConfig(name string, path string) (*StackConfig, error) {

	// make sure the file exists
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	if _, err := os.Stat(absPath); err != nil {
		log.Debugf("Stack file doesn't exist: %s", absPath)
		return nil, errors.WithStack(err)
	}

	yamlFile, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, errors.Wrapf(err, "Error reading YAML file %s", path)
	}

	loaded := map[string]interface{}{}

	err = yaml.Unmarshal(yamlFile, loaded)
	if err != nil {
		return nil, errors.Wrapf(err, "Error loading stack file %s", path)
	}

	log.Debugf("Loaded stack: %#v", loaded)

	stackConfig, ok := loaded[name]
	if !ok {
		return nil, errors.New(fmt.Sprintf("No stack called '%s' found in stack file %s", name, path))
	}

	log.Debugf("Loaded stack '%s' from file '%s'", name, path)

	stackConfigString, err := yaml.Marshal(stackConfig)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	log.Debugf("String stack config:\n%s", stackConfigString)

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

	err = yaml.Unmarshal(stackConfigString, &stack)
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
