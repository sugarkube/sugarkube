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

package templater

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/sugarkube/sugarkube/internal/pkg/log"
	"github.com/sugarkube/sugarkube/internal/pkg/utils"
	"os"
	"text/template"
)

var CustomFunctions = template.FuncMap{
	"exists":      exists,
	"findFiles":   findFiles,
	"mapPrintF":   mapPrintF,
	"listString":  listString,
	"isSet":       isSet,
	"removeEmpty": removeEmpty,
}

// Turn separate string parameters into a single []string array
func listString(elems ...string) []string {
	output := make([]string, 0)

	for _, elem := range elems {
		output = append(output, elem)
	}

	return output
}

// Runs sprintf over all elements of a list
func mapPrintF(pattern string, genericItems interface{}) []string {
	output := make([]string, 0)

	items, ok := genericItems.([]interface{})
	if ok {
		for _, item := range items {
			output = append(output, fmt.Sprintf(pattern, item))
		}
	} else {
		stringItems, ok := genericItems.([]string)
		if ok {
			for _, item := range stringItems {
				output = append(output, fmt.Sprintf(pattern, item))
			}
		}
	}

	return output
}

// Deletes empty elements of a list
func removeEmpty(genericItems interface{}) []string {
	output := make([]string, 0)

	items, ok := genericItems.([]interface{})
	if ok {
		for _, item := range items {
			if item != nil && item != "" {
				output = append(output, fmt.Sprintf("%v", item))
			}
		}
	} else {
		stringItems, ok := genericItems.([]string)
		if ok {
			for _, item := range stringItems {
				if item != "" {
					output = append(output, item)
				}
			}
		}
	}

	return output
}

// Checks whether a path exists. The filetype string determines whether to test for a file or dir
func exists(fileType string, path string) (bool, error) {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		} else {
			return false, err
		}
	}

	switch fileType {
	case "d":
		return info.IsDir(), nil
	case "f":
		return info.Mode().IsRegular(), nil
	case "any":
		return true, nil
	default:
		return true, errors.New("No filetype specified. See docs for the `exists` " +
			"template function.")
	}
}

// Takes a list of file names and searches an input path for them recursively.
// The result is a list of paths to files that exist matching the given patterns.
func findFiles(root string, patterns []string) ([]string, error) {

	output := make([]string, 0)

	for _, pattern := range patterns {
		filePaths, err := utils.FindFilesByPattern(root, pattern, true, false)
		if err != nil {
			return nil, errors.Wrapf(err, "Error finding '%s' in '%s'", pattern, root)
		}

		if len(filePaths) > 0 {
			log.Logger.Debugf("Found %d file(s) matching pattern '%s' under dir '%s': %s",
				len(filePaths), pattern, root, filePaths[0])

			for _, path := range filePaths {
				output = append(output, path)
			}
		} else {
			log.Logger.Tracef("No files found matching pattern '%s' under dir '%s'",
				pattern, root)
		}
	}

	return output, nil
}

// Returns whether a key is in a map (Sprig provides a similar function but it
// requires a map[string]interface{} and ours are map[interface{}]interface{}
func isSet(input map[interface{}]interface{}, key string) bool {
	_, ok := input[key]
	return ok
}
