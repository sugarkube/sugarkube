package vars

import (
	"github.com/pkg/errors"
	"github.com/sugarkube/sugarkube/internal/pkg/log"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
	"path/filepath"
)

type Stack struct {
	Name        string
	Provider    string
	Provisioner string
	Profile     string
	Cluster     string
	VarsFiles   []string
	Manifests   []string
}

// Loads a stack from a YAML file and returns it or an error
func LoadStack(name string, path string) (*Stack, error) {

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

	stackConfigString, err := yaml.Marshal(loaded[name])
	if err != nil {
		return nil, errors.WithStack(err)
	}

	log.Debugf("String stack config:\n%s", stackConfigString)

	stack := Stack{Name: name}
	err = yaml.Unmarshal(stackConfigString, &stack)

	log.Debugf("Loaded stack config: %#v", stack)

	return &stack, nil
}
