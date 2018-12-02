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
	"text/template"
)

var CustomFunctions = template.FuncMap{
	"findFiles": findFiles,
	"mapPrintF": mapPrintF,
}

// Runs sprintf over all elements of a list
func mapPrintF(pattern string, items []string) []string {
	output := make([]string, len(items))

	for i, item := range items {
		output[i] = fmt.Sprintf(pattern, item)
	}

	return output
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

		if len(filePaths) == 1 {
			log.Logger.Debugf("Found a file matching pattern '%s' under dir '%s': %s",
				pattern, root, filePaths[0])
			output = append(output, filePaths[0])
		} else if len(filePaths) > 1 {
			return nil, errors.New(fmt.Sprintf("Found multiple files matching pattern '%s' in '%s'. Don't "+
				"know which to choose...", pattern, root))
		} else {
			log.Logger.Debugf("No files found matching pattern '%s' under dir '%s'",
				pattern, root)
		}
	}

	return output, nil
}
