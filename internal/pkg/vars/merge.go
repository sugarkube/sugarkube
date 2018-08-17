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

		yamlFile, err := ioutil.ReadFile(path)
		if err != nil {
			log.Debug("yamlFile.Get err   #%v ", err)
		}

		var loaded = map[string]interface{}{}

		err = yaml.Unmarshal(yamlFile, loaded)
		if err != nil {
			log.Fatalf("Unmarshal: %v", err)
		}

		mergo.Merge(&result, loaded)
	}

	return &result
}
