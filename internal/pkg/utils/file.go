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

package utils

import (
	"github.com/pkg/errors"
	"github.com/sugarkube/sugarkube/internal/pkg/log"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Appends text to a file
func AppendToFile(filename string, text string) error {
	// create the file if it doesn't exist
	f, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0744)
	if err != nil {
		return errors.Wrapf(err, "Error opening file %s", filename)
	}

	defer f.Close()

	if _, err = f.WriteString(text); err != nil {
		return errors.Wrapf(err, "Error writing to file %s", filename)
	}

	return nil
}

// Search for files in a directory matching a regex, optionally recursively.
// If preferSymlinks is true, return paths will be replaced by symlinks where
// possible.
func FindFilesByPattern(rootDir string, pattern string, recursive bool,
	preferSymlinks bool) ([]string, error) {

	log.Logger.Debugf("Searching for files matching regex '%s' under dir '%s'",
		pattern, rootDir)

	re := regexp.MustCompile(pattern)
	results := make([]string, 0)

	links := make(map[string]string)

	if recursive {
		// todo - rewrite to support symlinks and excluding the .sugarkube cache directory
		err := filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return errors.WithStack(err)
			}

			// if the file is a symlink, save the destination so we can replace it later
			if info.Mode()&os.ModeSymlink != 0 {
				realPath, err := os.Readlink(path)
				if err != nil {
					return errors.Wrapf(err, "Error reading symlink '%s'", path)
				}

				links[realPath] = filepath.Base(path)
				return nil
			}

			if match := re.FindString(path); match != "" {
				results = append(results, path)
			}
			return nil
		})
		if err != nil {
			return nil, errors.WithStack(err)
		}

		if preferSymlinks && len(links) > 0 {
			// iterate through results replacing any paths that contain symlink
			// targets to be the symlinks themselves
			for i, result := range results {
				for linkTarget, link := range links {
					if strings.Contains(result, linkTarget) {
						// Too noisy. Commented out.
						//log.Logger.Debugf("Replacing link target '%s' with "+
						//	"link '%s' in result '%s'", linkTarget, link, result)
						results[i] = strings.Replace(result, linkTarget, link, 1)

						// verify that the updated path exists
						_, err := os.Stat(results[i])
						if err != nil {
							return nil, errors.Wrapf(err, "Path updated with "+
								"symlink '%s' doesn't exist", results[i])
						}
					}
				}
			}
		}

	} else {
		files, err := ioutil.ReadDir(rootDir)
		if err != nil {
			return nil, errors.WithStack(err)
		}

		for _, f := range files {
			if match := re.FindString(f.Name()); match != "" {
				results = append(results, filepath.Join(rootDir, match))
			}
		}
	}

	for i, result := range results {
		absResult, err := filepath.Abs(result)
		if err != nil {
			return nil, errors.WithStack(err)
		}

		results[i] = absResult
	}

	return results, nil
}

// Strips the extension from a file name
func StripExtension(path string) string {
	extension := filepath.Ext(path)
	return strings.TrimSuffix(path, extension)
}
