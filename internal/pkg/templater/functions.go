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
	"text/template"
)

var CustomFunctions = template.FuncMap{
	"findFiles": findFiles,
	"mapPrintF": mapPrintF,
}

// Runs sprintf over all elements of a list
func mapPrintF(pattern string, items []string) []string {
	output := make([]string, len(items))

	for _, item := range items {
		output = append(output, fmt.Sprintf(pattern, item))
	}

	return output
}

// Takes a list of file names and searches an input path for them recursively.
// The result is a list of paths to files that exist matching the given patterns.
func findFiles(pattern []string) []string {
	// todo - implement
	return pattern
}
