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

package utils

import (
	"encoding/json"
	"github.com/pkg/errors"
	"strconv"
)

// Creates a nested map with the final element an empty string
func BlankNestedMap(accumulator map[string]interface{}, elements []string) map[string]interface{} {
	if len(elements) == 1 {
		accumulator[elements[0]] = ""
		return accumulator
	} else {
		accumulator[elements[0]] = BlankNestedMap(map[string]interface{}{}, elements[1:])
		return accumulator
	}
}

// Performs a deep copy of an input by marshalling to and from JSON
func DeepCopy(in interface{}, out interface{}) error {

	bytes, err := json.Marshal(in)
	if err != nil {
		return errors.WithStack(err)
	}

	err = json.Unmarshal(bytes, out)
	if err != nil {
		return errors.Wrapf(err, "Error unmarshalling JSON")
	}

	return nil
}

// Returns true if all conditions are true. Conditions must be parseable as booleans.
func All(conditions []string) (bool, error) {
	var boolCondition bool
	var err error
	for _, condition := range conditions {
		boolCondition, err = strconv.ParseBool(condition)
		if err != nil {
			return false, errors.WithStack(err)
		}

		if !boolCondition {
			return false, nil
		}
	}

	return true, nil
}
