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

package installable

import (
	"bytes"
	"fmt"
	"github.com/pkg/errors"
	"github.com/sugarkube/sugarkube/internal/pkg/acquirer"
	"github.com/sugarkube/sugarkube/internal/pkg/constants"
	"github.com/sugarkube/sugarkube/internal/pkg/log"
	"github.com/sugarkube/sugarkube/internal/pkg/structs"
	"github.com/sugarkube/sugarkube/internal/pkg/templater"
	"github.com/sugarkube/sugarkube/internal/pkg/utils"
	"gopkg.in/yaml.v2"
	"strings"
)

type Kapp struct {
	descriptor structs.KappDescriptor
	manifestId string
	state      string
	config     structs.KappConfig
}

// Returns the non-fully qualified ID
func (k Kapp) Id() string {
	return k.descriptor.Id
}

// Returns the manifest ID
func (k Kapp) ManifestId() string {
	return k.manifestId
}

func (k Kapp) State() string {
	return k.state
}

func (k Kapp) PostActions() []string {
	return k.descriptor.PostActions
}

// Returns the fully-qualified ID of a kapp
func (k Kapp) FullyQualifiedId() string {
	if k.manifestId == "" {
		return k.Id()
	} else {
		return strings.Join([]string{k.manifestId, k.Id()}, constants.NamespaceSeparator)
	}
}

// Returns an array of acquirers configured for the sources for this kapp. We need to recompute these each time
// instead of caching them so that any manifest overrides will take effect.
func (k Kapp) Acquirers() ([]acquirer.Acquirer, error) {
	acquirers, err := acquirer.GetAcquirersFromSources(k.descriptor.Sources)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return acquirers, nil
}

// (Re)loads the kapp's sugarkube.yaml file, templates it and saves is at as an attribute on the kapp
func (k *Kapp) RefreshConfig(templateVars map[string]interface{}) error {

	configFilePaths, err := utils.FindFilesByPattern(k.CacheDir(), constants.KappConfigFileName,
		true, false)
	if err != nil {
		return errors.Wrapf(err, "Error finding '%s' in '%s'",
			constants.KappConfigFileName, k.CacheDir())
	}

	if len(configFilePaths) == 0 {
		return errors.New(fmt.Sprintf("No '%s' file found for kapp "+
			"'%s' in %s", constants.KappConfigFileName, k.FullyQualifiedId(), k.CacheDir()))
	} else if len(configFilePaths) > 1 {
		// todo - have a way of declaring the 'right' one in the manifest
		panic(fmt.Sprintf("Multiple '%s' found for kapp '%s'. Disambiguation "+
			"not implemented yet: %s", constants.KappConfigFileName, k.FullyQualifiedId(),
			strings.Join(configFilePaths, ", ")))
	}

	configFilePath := configFilePaths[0]

	var outBuf bytes.Buffer
	err = templater.TemplateFile(configFilePath, &outBuf, templateVars)
	if err != nil {
		return errors.WithStack(err)
	}

	log.Logger.Tracef("Rendered %s file at '%s' to: \n%s", constants.KappConfigFileName,
		configFilePath, outBuf.String())

	config := structs.KappConfig{}
	err = yaml.Unmarshal(outBuf.Bytes(), &config)
	if err != nil {
		return errors.Wrapf(err, "Error unmarshalling rendered %s file: %s",
			constants.KappConfigFileName, outBuf.String())
	}

	k.config = config
	return nil
}
