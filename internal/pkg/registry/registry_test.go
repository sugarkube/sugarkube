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
	"github.com/stretchr/testify/assert"
	"github.com/sugarkube/sugarkube/internal/pkg/log"
	"testing"
)

func init() {
	log.ConfigureLogger("trace", false)
}

func TestSimple(t *testing.T) {
	tests := []struct {
		key   string
		value interface{}
	}{
		{"tstStr", "someVal"},
		{"tstMap", map[string]interface{}{
			"x": 1,
			"y": 5,
		}},
		{"missing", nil},
	}

	registry := New()

	for _, test := range tests {
		if test.value != nil {
			registry.Set(test.key, test.value)
		}

		expectedOk := test.value != nil
		actualValue, actualOk := registry.Get(test.key)

		assert.Equal(t, expectedOk, actualOk)

		if expectedOk {
			assert.Equal(t, test.value, actualValue)
		}
	}
}

func TestNestedMap(t *testing.T) {
	registry := New()

	// the submap we expect under the 'tests' key
	expectedTests := map[string]interface{}{
		"testMap": map[string]interface{}{
			"val": "tst",
			"num": 2,
			"dot": map[string]interface{}{
				"ted": "xyz",
				"sed": "xxyz",
			},
		},
	}

	inputData := map[string]interface{}{
		"val":     "tst",
		"num":     2,
		"dot.ted": "xyz",
		"dot.sed": "xxyz",
	}

	registry.Set("tests.testMap", inputData)

	// get the whole map
	val, ok := registry.Get("tests")
	assert.Equal(t, expectedTests, val)
	assert.True(t, ok)

	val, ok = registry.Get("tests.testMap")
	assert.Equal(t, expectedTests["testMap"], val)
	assert.True(t, ok)

	// get the value of a key in the map
	val, ok = registry.Get("tests.testMap.val")
	assert.Equal(t, (expectedTests["testMap"].(map[string]interface{}))["val"], val)
	assert.True(t, ok)

	val, ok = registry.Get("tests.testMap.num")
	assert.Equal(t, 2, val)
	assert.True(t, ok)

	val, ok = registry.Get("tests.testMap.dot.ted")
	assert.Equal(t, "xyz", val)
	assert.True(t, ok)

	// dotted values are converted to maps, so we should be able to set new values on it
	registry.Set("tests.testMap.dot.new", 123)
	val, ok = registry.Get("tests.testMap.dot.new")
	assert.Equal(t, 123, val)
	assert.True(t, ok)

	// and get the whole map
	val, ok = registry.Get("tests.testMap.dot")
	assert.Equal(t, map[string]interface{}{"ted": "xyz", "sed": "xxyz", "new": 123}, val)
	assert.True(t, ok)

	// try to get a non-existent value
	val, ok = registry.Get("tests.testMap.missing")
	assert.False(t, ok)
}

func TestDelete(t *testing.T) {
	registry := New()

	innerMap := map[string]interface{}{
		"val": "tst",
		"num": 2,
		"dot": map[string]interface{}{
			"ted": "xyz",
			"sed": "xxyz",
		},
	}

	inputData := map[string]interface{}{
		"val":     "tst",
		"num":     2,
		"dot.ted": "xyz",
		"dot.sed": "xxyz",
	}

	registry.Set("tests.testMap", inputData)

	// make sure we get the expected data
	val, ok := registry.Get("tests.testMap")
	assert.Equal(t, innerMap, val)
	assert.True(t, ok)

	// make sure we can delete single values
	registry.Delete("tests.testMap.dot.sed")
	val, ok = registry.Get("tests.testMap")
	assert.Equal(t, map[string]interface{}{
		"val": "tst",
		"num": 2,
		"dot": map[string]interface{}{
			"ted": "xyz",
		},
	}, val)
	assert.True(t, ok)

	// make sure we can delete entire submaps
	registry.Delete("tests.testMap.dot")
	val, ok = registry.Get("tests.testMap")
	assert.Equal(t, map[string]interface{}{
		"val": "tst",
		"num": 2,
	}, val)
	assert.True(t, ok)
}
