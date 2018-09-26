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

package kapp

import (
	"github.com/pkg/errors"
	"github.com/sugarkube/sugarkube/internal/pkg/log"
	"github.com/sugarkube/sugarkube/internal/pkg/utils"
	"os"
	"path/filepath"
	"strings"
)

// This searches a directory tree from a given root path for files whose values
// should be merged together for a kapp based on the values of the stack config
// and the kapp itself.
func FindKappVarsFiles(stackConfig *StackConfig, kappObj *Kapp) ([]string, error) {
	validNames := []string{
		stackConfig.Name,
		stackConfig.Provider,
		stackConfig.Provisioner,
		stackConfig.Account,
		stackConfig.Region,
		stackConfig.Profile,
		stackConfig.Cluster,
		kappObj.Id,
	}

	for _, acquirerObj := range kappObj.Sources {
		validNames = append(validNames, acquirerObj.Name())

		id, err := acquirerObj.Id()
		if err != nil {
			return nil, errors.WithStack(err)
		}

		validNames = append(validNames, id)
	}

	paths := make([]string, 0)

	for _, searchDir := range stackConfig.KappVarsDirs {
		searchPath, err := filepath.Abs(filepath.Join(stackConfig.Dir(), searchDir))
		if err != nil {
			return nil, errors.WithStack(err)
		}

		log.Logger.Debugf("Searching for files/dirs under '%s' with basenames: %s",
			searchPath, strings.Join(validNames, ", "))

		err = filepath.Walk(searchPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return errors.WithStack(err)
			}

			log.Logger.Debugf("Visiting: %s", path)

			if info.IsDir() {
				if utils.InStringArray(validNames, info.Name()) || info.Name() == filepath.Base(searchPath) {
					log.Logger.Debugf("Will search kapp var path: %s", path)
					return nil
				} else {
					log.Logger.Debugf("Skipping kapp var dir: %s", path)
					return filepath.SkipDir
				}
			} else {
				basename := filepath.Base(path)
				ext := filepath.Ext(basename)

				if strings.ToLower(ext) != ".yaml" {
					log.Logger.Debugf("Ignoring non-yaml file: %s", path)
					return nil
				}

				nakedBasename := strings.Replace(basename, ext, "", 1)

				if basename == "values.yaml" || utils.InStringArray(validNames, nakedBasename) {
					log.Logger.Debugf("Adding kapp var file: %s", path)
					// prepend the value to the array to maintain ordering
					paths = append([]string{path}, paths...)
				}
			}

			return nil
		})

		if err != nil {
			return nil, errors.WithStack(err)
		}
	}

	log.Logger.Debugf("Kapp var paths for kapp '%s' are: %s", kappObj.Id,
		strings.Join(paths, ", "))

	return paths, nil
}
