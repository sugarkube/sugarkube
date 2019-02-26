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
	"bytes"
	"fmt"
	"github.com/pkg/errors"
	"github.com/sugarkube/sugarkube/internal/pkg/log"
	"github.com/sugarkube/sugarkube/internal/pkg/templater"
	"github.com/sugarkube/sugarkube/internal/pkg/utils"
	"gopkg.in/yaml.v2"
	"strings"
)

const CONFIG_FILE = "sugarkube.yaml"

// Loads the kapp's sugarkube.yaml file, renders it and sets its attributes as
// an attribute on the kapp
func (k *Kapp) Load(mergedKappVars map[string]interface{}) error {

	configFilePaths, err := utils.FindFilesByPattern(k.CacheDir(), CONFIG_FILE,
		true, false)
	if err != nil {
		return errors.Wrapf(err, "Error finding '%s' in '%s'",
			CONFIG_FILE, k.CacheDir())
	}

	if len(configFilePaths) == 0 {
		return errors.New(fmt.Sprintf("No '%s' file found for kapp "+
			"'%s' in %s", CONFIG_FILE, k.FullyQualifiedId(), k.CacheDir()))
	} else if len(configFilePaths) > 1 {
		// todo - have a way of declaring the 'right' one in the manifest
		panic(fmt.Sprintf("Multiple '%s' found for kapp '%s'. Disambiguation "+
			"not implemented yet: %s", CONFIG_FILE, k.FullyQualifiedId(),
			strings.Join(configFilePaths, ", ")))
	}

	configFilePath := configFilePaths[0]

	var outBuf bytes.Buffer
	err = templater.TemplateFile(configFilePath, &outBuf, mergedKappVars)
	if err != nil {
		return errors.WithStack(err)
	}

	log.Logger.Debugf("Rendered %s file at '%s' to: \n%s", CONFIG_FILE,
		configFilePath, outBuf.String())

	config := Config{}
	err = yaml.Unmarshal(outBuf.Bytes(), &config)
	if err != nil {
		return errors.Wrapf(err, "Error unmarshalling rendered %s file: %s",
			CONFIG_FILE, outBuf.String())
	}

	k.Config = config
	return nil
}
