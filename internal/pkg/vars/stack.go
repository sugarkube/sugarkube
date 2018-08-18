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

type StackConfig struct {
	Name          string
	Provider      string
	Provisioner   string
	Profile       string
	Cluster       string
	VarsFilesDirs []string `yaml:"vars"`
	Manifests     []string
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

	stack := StackConfig{Name: name}

	err = yaml.Unmarshal(stackConfigString, &stack)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	log.Debugf("Loaded stack config: %#v", stack)

	return &stack, nil
}

// Parses and merges all specified vars files and returns the groups of vars
func (s *StackConfig) Vars() map[string]interface{} {
	for _, varFile := range s.VarsFilesDirs {
		groupedFiles := GroupFiles(varFile)
		log.Debugf("Grouped: %#v", groupedFiles)
	}

	vars := map[string]interface{}{}

	return vars
}
