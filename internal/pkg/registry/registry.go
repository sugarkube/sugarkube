/*
 * Copyright 2019 The Sugarkube Authors
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

package registry

import (
	"github.com/pkg/errors"
	"github.com/sugarkube/sugarkube/internal/pkg/constants"
	"github.com/sugarkube/sugarkube/internal/pkg/convert"
	"github.com/sugarkube/sugarkube/internal/pkg/interfaces"
	"github.com/sugarkube/sugarkube/internal/pkg/log"
	"os"
	"reflect"
	"strings"
)

// A registry so that different parts of the program can set and access values
type Registry struct {
	data map[string]interface{}
}

func New() interfaces.IRegistry {
	// todo - find a better way of initialising this. We need to do this
	//  so `kapp vars` doesn't output '<no value>' which might be confusing.
	kubeConfig := os.Getenv(strings.ToUpper(constants.RegistryKeyKubeConfig))

	return &Registry{
		data: map[string]interface{}{
			constants.RegistryKeyKubeConfig: kubeConfig,
		},
	}
}

// Returns a copy of the registry
func (r *Registry) Copy() (interfaces.IRegistry, error) {
	newRegistry := New()
	for k, v := range r.AsMap() {
		err := newRegistry.Set(k, v)
		if err != nil {
			return nil, errors.WithStack(err)
		}
	}

	return newRegistry, nil
}

// Add data to the registry.
func (r *Registry) Set(key string, value interface{}) error {
	log.Logger.Tracef("Setting registry key='%s' to value=%+v", key, value)
	data, err := nestedMap(r.data, strings.Split(key, constants.RegistryFieldSeparator), value)
	if err != nil {
		return errors.WithStack(err)
	}
	r.data = data
	log.Logger.Tracef("Set registry data to: %+v", r.data)
	return nil
}

// Inserts the given value into the data map. `elements` is a list of map keys - if the map for any
// particular key doesn't exist a blank map will be created.
func nestedMap(data map[string]interface{}, elements []string, value interface{}) (map[string]interface{}, error) {

	// too verbose even for tracing, so comment it out for now
	//log.Logger.Tracef("new iteration: data=%v, elements=%v, value=%+v", data, elements, value)

	key := elements[0]

	if len(elements) == 1 {
		reflected := reflect.ValueOf(value)
		if reflected.Kind() == reflect.Map {
			//log.Logger.Tracef("Value is a map: %v", value)

			itemMap := getMapOrNew(data, key)

			valueMap, ok := value.(map[string]interface{})
			if !ok {
				var err error
				valueMap, err = convert.MapInterfaceInterfaceToMapStringInterface(
					value.(map[interface{}]interface{}))
				if err != nil {
					return nil, errors.WithStack(err)
				}
			}
			//log.Logger.Tracef("valueMap=%v", valueMap)

			// if the value is a map, run each key through this function to split dotted keys
			for k, v := range valueMap {
				kParts := strings.Split(k, constants.RegistryFieldSeparator)
				//log.Logger.Tracef("Branch 1: Running with: itemMap=%v kParts=%v, v=%v", itemMap, kParts, v)
				result, err := nestedMap(itemMap, kParts, v)
				if err != nil {
					return nil, errors.WithStack(err)
				}
				data[key] = result

			}
		} else {
			//log.Logger.Tracef("Finally setting value of key=%s to '%v' in map %v", key, value, data)
			data[key] = value
		}

		return data, nil
	} else {
		// if the map exists fetch it, otherwise create it
		itemMap := getMapOrNew(data, key)
		//log.Logger.Tracef("Branch 2: elements=%v value=%v", elements[1:], value)
		result, err := nestedMap(itemMap, elements[1:], value)
		if err != nil {
			return nil, errors.WithStack(err)
		}
		data[key] = result
		return data, nil
	}
}

// Gets a submap from a map, or returns a new map if there is no submap
func getMapOrNew(data map[string]interface{}, key string) map[string]interface{} {
	_, ok := data[key]
	if ok {
		return data[key].(map[string]interface{})
	} else {
		return map[string]interface{}{}
	}
}

// Get value from the registry. `constants.RegistryFieldSeparator` is used to separate the key into submaps
func (r *Registry) Get(key string) (interface{}, bool) {
	return nestedLookup(r.data, strings.Split(key, constants.RegistryFieldSeparator))
}

// Gets a value from a nested map. Also returns a boolean indicating whether the value was found in
// the map
func nestedLookup(data map[string]interface{}, elements []string) (interface{}, bool) {
	key := elements[0]

	if len(elements) == 1 {
		val, ok := data[key]
		if !ok {
			return nil, false
		}
		return val, true
	} else {
		// see if the key is in the map
		subMap, ok := data[key]
		if ok {
			// yes, so recurse into the submap
			return nestedLookup(subMap.(map[string]interface{}), elements[1:])
		} else {
			// not found
			return nil, false
		}
	}
}

// Return the registry as a map
func (r Registry) AsMap() map[string]interface{} {
	return r.data
}

// Delete a key from the registry. If the key contains `constants.RegistryFieldSeparator`, submaps will
// be traversed
func (r *Registry) Delete(key string) {
	log.Logger.Tracef("Deleting key='%s' from the registry", key)
	nestedDelete(r.data, strings.Split(key, constants.RegistryFieldSeparator))
	log.Logger.Tracef("Registry data after deletion is: %+v", r.data)
}

// Deletes a value from a map, traversing submaps as necessary
func nestedDelete(data map[string]interface{}, elements []string) {
	key := elements[0]

	item, ok := data[key]
	if !ok {
		return
	}

	if len(elements) == 1 {
		delete(data, key)
	} else {
		nestedDelete(item.(map[string]interface{}), elements[1:])
		return
	}
}
