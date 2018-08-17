package vars

import (
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
	"path/filepath"
)

type Stack struct {
	name        string
	provider    string
	provisioner string
	profile     string
	cluster     string
	varsFiles   []string
	manifests   []string
}

// Loads a stack from a YAML file and returns it or an error
func LoadStack(name string, path string) (*Stack, error) {

	// make sure the file exists
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	if _, err := os.Stat(absPath); err != nil {
		return nil, errors.WithStack(err)
	}

	yamlFile, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, errors.Wrapf(err, "Error reading YAML file %s", path)
	}

	stack := Stack{name: name}
	err = yaml.Unmarshal(yamlFile, &stack)
	if err != nil {
		errors.Wrapf(err, "Error loading stack file %s", path)
	}

	return &stack, nil
}
