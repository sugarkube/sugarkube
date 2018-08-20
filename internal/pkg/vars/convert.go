package vars

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/sugarkube/sugarkube/internal/pkg/log"
	"reflect"
)

// Return an error if the type of an input can't easily be converted
func convertStringable(input interface{}) (string, error) {

	vKind := reflect.TypeOf(input).Kind()

	if vKind == reflect.Array || vKind == reflect.Slice ||
		vKind == reflect.Struct || vKind == reflect.Map {
		return "", errors.New(
			fmt.Sprintf("Can't convert array/slice/struct/map value: %#v", input))
	}

	return fmt.Sprintf("%v", input), nil
}

// Converts a map with keys and values as interfaces to a map with keys and values as strings
func InterfaceMapToStringMap(input map[interface{}]interface{}) (map[string]string, error) {

	log.Debugf("Converting map of interfaces to map of strings. Input=%#v", input)

	output := make(map[string]string)

	for k, v := range input {
		strKey, err := convertStringable(k)
		if err != nil {
			return nil, errors.WithStack(err)
		}

		strVal, err := convertStringable(v)
		if err != nil {
			return nil, errors.WithStack(err)
		}

		output[strKey] = strVal
	}

	log.Debugf("Converted map of interfaces to map of strings. Output=%#v", output)

	return output, nil
}
