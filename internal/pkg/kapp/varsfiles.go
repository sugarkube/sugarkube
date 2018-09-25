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
func FindKappVarsFiles(rootPath string, stackConfig *StackConfig, kappObj *Kapp) ([]string, error) {
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

	log.Logger.Debugf("Searching for files/dirs under '%s' with basenames: %s",
		rootPath, strings.Join(validNames, ", "))

	paths := make([]string, 0)

	err := filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return errors.WithStack(err)
		}

		log.Logger.Debugf("Visiting: %s", info.Name())

		if info.IsDir() {
			if utils.InStringArray(validNames, info.Name()) {
				log.Logger.Debugf("Adding kapp var path: %s", info.Name())
				paths = append(paths, info.Name())
			} else {
				log.Logger.Debugf("Skipping kapp var dir: %s", info.Name())
				return filepath.SkipDir
			}
		} else {
			basename := filepath.Base(info.Name())
			ext := filepath.Ext(basename)

			if strings.ToLower(ext) != "yaml" {
				return filepath.SkipDir
			}

			if basename == "values.yaml" || utils.InStringArray(validNames, basename) {
				log.Logger.Debugf("Adding kapp var file: %s", basename)
				paths = append(paths, basename)
			}
		}

		return nil
	})

	if err != nil {
		return nil, errors.WithStack(err)
	}

	log.Logger.Debugf("Kapp var paths for kapp '%s' are: %s", kappObj.Id,
		strings.Join(paths, ", "))

	return paths, nil
}
