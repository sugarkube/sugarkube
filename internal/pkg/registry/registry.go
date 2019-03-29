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
	"github.com/sugarkube/sugarkube/internal/pkg/constants"
	"github.com/sugarkube/sugarkube/internal/pkg/convert"
	"github.com/sugarkube/sugarkube/internal/pkg/log"
	"strings"
)

// A registry so that different parts of the program can set and access values
type Registry struct {
	mapStringString map[string]string
}

func New() Registry {
	return Registry{
		mapStringString: map[string]string{
			// todo - find a better way of initialising this. We need to do this
			//  so `kapp vars` doesn't output '<no value>' which might be confusing.
			//  Maybe we should try to pull these from env vars?
			constants.RegistryKeyKubeConfig: "",
		},
	}
}

// Add a string to the registry.
// *Note* For now there's a limitation where string keys mustn't contain dot
// characters because they won't be merged with stack config vars correctly
// (e.g. as a map). So for now to avoid unpredictable behaviour we don't permit
// keys with dots at all.
func (r *Registry) SetString(key string, value string) {
	if strings.Contains(key, ".") {
		log.Logger.Fatalf("Keys with dots ('.') are not currently merged " +
			"as a map, Only top-level values can be stored in the registry")
	}
	r.mapStringString[key] = value
}

// Get a string from the registry
func (r *Registry) GetString(key string) (string, bool) {
	val, ok := r.mapStringString[key]
	if !ok {
		return "", false
	}

	return val, true
}

func (r Registry) AsMap() map[string]interface{} {
	return convert.MapStringStringToMapStringInterface(r.mapStringString)
}
