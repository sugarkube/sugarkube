/*
 * Copyright 2018 The Sugarkube Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package convert

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/sugarkube/sugarkube/internal/pkg/acquirer"
	"github.com/sugarkube/sugarkube/internal/pkg/log"
	"github.com/sugarkube/sugarkube/internal/pkg/structs"
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

// Converts a map with keys and values as interfaces to a map with keys and values as strings or
// returns an error if types can't be sanely converted
func MapInterfaceInterfaceToMapStringString(input map[interface{}]interface{}) (map[string]string, error) {

	log.Logger.Tracef("Converting map of interfaces to map of strings. Input=%#v", input)

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

	log.Logger.Tracef("Converted map of interfaces to map of strings. Output=%#v", output)

	return output, nil
}

// Converts a map with keys and values as interfaces to a map with string keys and values unchanged or
// returns an error if types can't be sanely converted
func MapInterfaceInterfaceToMapStringInterface(input map[interface{}]interface{}) (map[string]interface{}, error) {

	log.Logger.Debugf("Converting map of interfaces to map with string keys. Input=%#v", input)

	output := make(map[string]interface{})

	for k, v := range input {
		strKey, err := convertStringable(k)
		if err != nil {
			return nil, errors.WithStack(err)
		}

		output[strKey] = v
	}

	log.Logger.Debugf("Converted map of interfaces to map with string keys. Output=%#v", output)

	return output, nil
}

// Converts a map[string]string to a map[string]interface{}
func MapStringStringToMapStringInterface(input map[string]string) map[string]interface{} {
	output := map[string]interface{}{}
	for k, v := range input {
		output[k] = v
	}

	return output
}

// Converts a map[string]interface{} to a map[string]string
//func InterfaceToMapStringString(input interface{}) map[string]string {
//	output := map[string]string{}
//	for k, v := range input.(map[string]interface{}) {
//		output[k] = v.(string)
//	}
//
//	return output
//}

// Converts a KappDescriptorWithLists to a KappDescriptorWithMaps
func KappDescriptorWithListsToMap(descriptor structs.KappDescriptorWithLists) (
	structs.KappDescriptorWithMaps, error) {
	// convert the descriptor to be a KappDescriptorWithMaps and set it as initial config layer
	sources := make(map[string]structs.Source, 0)

	// we need to get the ID from the acquirer associated with the source
	for _, source := range descriptor.Sources {
		acquirerObj, err := acquirer.New(source)
		if err != nil {
			return structs.KappDescriptorWithMaps{}, errors.WithStack(err)
		}
		sources[acquirerObj.Id()] = source
	}

	outputs := make(map[string]structs.Output, 0)

	for _, output := range descriptor.Outputs {
		outputs[output.Id] = output
	}

	return structs.KappDescriptorWithMaps{
		Id:         descriptor.Id,
		KappConfig: descriptor.KappConfig,
		Sources:    sources,
		Outputs:    outputs,
	}, nil
}
