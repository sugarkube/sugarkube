package vars

import (
	"fmt"
	"github.com/sugarkube/sugarkube/internal/pkg/log"
)

// Converts a map with keys and values as interfaces to a map with keys and values as strings
func InterfaceMapToStringMap(input map[interface{}]interface{}) map[string]string {

	log.Debugf("Converting map of interfaces to map of strings. Input=%#v", input)

	output := make(map[string]string)

	for k, v := range input {
		strKey := fmt.Sprintf("%v", k)
		strValue := fmt.Sprintf("%v", v)

		output[strKey] = strValue
	}

	log.Debugf("Converted map of interfaces to map of strings. Output=%#v", output)

	return output
}
