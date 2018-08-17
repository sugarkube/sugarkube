package vars

import (
	"github.com/imdario/mergo"
	"github.com/sugarkube/sugarkube/internal/pkg/log"
	"gopkg.in/yaml.v2"
	"io/ioutil"
)

func Merge(paths ...string) *map[string]interface{} {

	result := map[string]interface{}{}

	for _, path := range paths {

		log.Debug("Loading path", path)

		yamlFile, err := ioutil.ReadFile(path)
		if err != nil {
			// todo - raise an error; structured logging?
			log.Fatalf("Error reading YAML file: %v ", err)
		}

		var loaded = map[string]interface{}{}

		err = yaml.Unmarshal(yamlFile, loaded)
		if err != nil {
			// todo - raise an error; structured logging?
			log.Fatalf("Error loading YAML: %v", err)
		}

		log.Debugf("Merging %v with %v", result, loaded)

		mergo.Merge(&result, loaded, mergo.WithOverride)
	}

	return &result
}
