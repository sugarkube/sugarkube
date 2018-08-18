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
	FilePath      string
	Provider      string
	Provisioner   string
	Profile       string
	Cluster       string
	VarsFilesDirs []string `yaml:"vars"`
	Manifests     []string
}

const valuesFile = "values.yaml"

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
func (s *StackConfig) dir() string {
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

// Searches for values.yaml files in configured directories and returns the
// result of merging them.
func (s *StackConfig) Vars() map[string]interface{} {
	vars := map[string]interface{}{}

	for _, varFile := range s.VarsFilesDirs {
		varDir := filepath.Join(s.dir(), varFile)
		groupedFiles := GroupFiles(varDir)

		valuesPaths, ok := groupedFiles[valuesFile]
		if !ok {
			log.Debugf("Skipping loading vars from directory %s. No %s file",
				varDir, valuesFile)
			continue
		}

		for _, path := range valuesPaths {
			Merge(&vars, path)
		}
	}

	return vars
}
