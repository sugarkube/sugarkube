package vars

import (
	"github.com/pkg/errors"
	"github.com/sugarkube/sugarkube/internal/pkg/log"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
	"path/filepath"
)

func LoadYamlFile(path string) (map[string]interface{}, error) {
	// make sure the file exists
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	if _, err := os.Stat(absPath); err != nil {
		log.Debugf("YAML file doesn't exist: %s", absPath)
		return nil, errors.WithStack(err)
	}

	yamlData, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, errors.Wrapf(err, "Error reading YAML file %s", path)
	}

	data := map[string]interface{}{}

	err = yaml.Unmarshal(yamlData, data)
	if err != nil {
		return nil, errors.Wrapf(err, "Error loading YAML file %s", path)
	}

	log.Debugf("YAML file: %#v", data)

	return data, nil
}
