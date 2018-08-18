package vars

import (
	"github.com/imdario/mergo"
	"github.com/pkg/errors"
	"github.com/sugarkube/sugarkube/internal/pkg/log"
	"gopkg.in/yaml.v2"
	"io/ioutil"
)

func Merge(result *map[string]interface{}, paths ...string) error {

	for _, path := range paths {

		log.Debug("Loading path", path)

		yamlFile, err := ioutil.ReadFile(path)
		if err != nil {
			return errors.Wrapf(err, "Error reading YAML file %s", path)
		}

		var loaded = map[string]interface{}{}

		err = yaml.Unmarshal(yamlFile, loaded)
		if err != nil {
			return errors.Wrapf(err, "Error loading YAML file: %s", path)
		}

		log.Debugf("Merging %v with %v", result, loaded)

		mergo.Merge(result, loaded, mergo.WithOverride)
	}

	return nil
}
