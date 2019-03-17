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

const ConfigFile = "sugarkube.yaml"

// Loads the kapp's sugarkube.yaml file, renders it and sets its attributes as
// an attribute on the kapp
func (k *Kapp) Load(mergedKappVars map[string]interface{}) error {

	configFilePaths, err := utils.FindFilesByPattern(k.CacheDir(), ConfigFile,
		true, false)
	if err != nil {
		return errors.Wrapf(err, "Error finding '%s' in '%s'",
			ConfigFile, k.CacheDir())
	}

	if len(configFilePaths) == 0 {
		return errors.New(fmt.Sprintf("No '%s' file found for kapp "+
			"'%s' in %s", ConfigFile, k.FullyQualifiedId(), k.CacheDir()))
	} else if len(configFilePaths) > 1 {
		// todo - have a way of declaring the 'right' one in the manifest
		panic(fmt.Sprintf("Multiple '%s' found for kapp '%s'. Disambiguation "+
			"not implemented yet: %s", ConfigFile, k.FullyQualifiedId(),
			strings.Join(configFilePaths, ", ")))
	}

	configFilePath := configFilePaths[0]

	var outBuf bytes.Buffer
	err = templater.TemplateFile(configFilePath, &outBuf, mergedKappVars)
	if err != nil {
		return errors.WithStack(err)
	}

	log.Logger.Debugf("Rendered %s file at '%s' to: \n%s", ConfigFile,
		configFilePath, outBuf.String())

	config := Config{}
	err = yaml.Unmarshal(outBuf.Bytes(), &config)
	if err != nil {
		return errors.Wrapf(err, "Error unmarshalling rendered %s file: %s",
			ConfigFile, outBuf.String())
	}

	k.Config = config
	return nil
}

// Templates global program configs for programs used by the kapp and merges
// them wtih the kapps own config declared in its sugarkube.yaml file
// todo - uncomment and test once viper supports not lowercasing all keys
// in config files on loading. See https://github.com/spf13/viper/pull/635
//func (k Kapp) MergeProgramConfigs(programConfigs map[string]program.Config,
//	mergedKappVars map[string]interface{}) (*Config, error) {
//	var err error
//	mergedConfig := &program.Config{}
//
//	// merge the defaults together for the programs used in the kapp
//	for _, programName := range k.Config.Requires {
//		defaultConfig, ok := programConfigs[programName]
//		if !ok {
//			log.Logger.Debugf("No global program config for program '%s'",
//				programName)
//			continue
//		}
//
//		yamlDefault, err := defaultConfig.AsYaml()
//		if err != nil {
//			return nil, errors.WithStack(err)
//		}
//
//		// template the default config
//		templatedConfigStr, err := templater.RenderTemplate(yamlDefault, mergedKappVars)
//		if err != nil {
//			return nil, errors.WithStack(err)
//		}
//
//		// convert it back to a program config
//		templatedConfig := program.Config{}
//		err = yaml.Unmarshal([]byte(templatedConfigStr), &templatedConfig)
//		if err != nil {
//			return nil, errors.WithStack(err)
//		}
//
//		err = mergo.Merge(mergedConfig, templatedConfig, mergo.WithOverride)
//		if err != nil {
//			return nil, errors.WithStack(err)
//		}
//	}
//
//	// finally merge the kapp's own config over the top
//	err = mergo.Merge(mergedConfig, k.Config.Config, mergo.WithOverride)
//	if err != nil {
//		return nil, errors.WithStack(err)
//	}
//
//	finalConfig := Config{
//		Requires: k.Config.Requires,
//		Config:   *mergedConfig,
//	}
//
//	return &finalConfig, nil
//}
