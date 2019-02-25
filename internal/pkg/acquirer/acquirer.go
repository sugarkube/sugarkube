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

package acquirer

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/sugarkube/sugarkube/internal/pkg/convert"
	"github.com/sugarkube/sugarkube/internal/pkg/log"
	"strings"
)

type Acquirer interface {
	acquire(dest string) error
	FullyQualifiedId() (string, error)
	Id() string
	Path() string
	IncludeValues() bool // todo - clarify if this is actually used, and if not, remove it
}

type Source struct {
	Id            string
	Uri           string
	IncludeValues bool // todo - see if we actually need this
}

const ACQUIRER_KEY = "acquirer"
const ID_KEY = "id"
const URI_KEY = "uri"

// Factory that creates acquirers
func acquirerFactory(name string, settings map[string]string) (Acquirer, error) {
	log.Logger.Debugf("Returning new %s acquirer", name)

	if name == GIT_ACQUIRER {
		acquirerObj, err := NewGitAcquirer(settings[ID_KEY], settings[URI_KEY], settings[BRANCH_KEY],
			settings[PATH_KEY], settings[INCLUDE_VALUES_KEY])
		if err != nil {
			return nil, errors.WithStack(err)
		}
		return acquirerObj, nil

	} else if name == FILE_ACQUIRER {
		acquirerObj, err := NewFileAcquirer(settings[ID_KEY], settings[URI_KEY])
		if err != nil {
			return nil, errors.WithStack(err)
		}
		return acquirerObj, nil
	}

	return nil, errors.New(fmt.Sprintf("Acquirer '%s' doesn't exist", name))
}

// Identifies the acquirer based on its settings, and returns a new instance of it
func NewAcquirer(settings map[string]string) (Acquirer, error) {
	// perhaps the acquirer is explicitly declared in settings
	acquirer := settings[ACQUIRER_KEY]

	uri := settings[URI_KEY]

	if strings.Contains(uri, ".git") || acquirer == GIT_ACQUIRER {
		return acquirerFactory(GIT_ACQUIRER, settings)
	}

	return nil, errors.New(fmt.Sprintf("Couldn't identify acquirer for URI '%s'", uri))
}

// Delegate to an acquirer implementation
func Acquire(a Acquirer, dest string) error {
	return a.acquire(dest)
}

// Parse acquirers from a values map
func ParseAcquirers(acquirerMaps []map[interface{}]interface{}) ([]Acquirer, error) {
	acquirers := make([]Acquirer, 0)
	// now we have a list of sources, get the acquirer for each one
	for _, acquirerMap := range acquirerMaps {
		acquirerStringMap, err := convert.MapInterfaceInterfaceToMapStringString(acquirerMap)
		if err != nil {
			return nil, errors.WithStack(err)
		}

		acquirerImpl, err := NewAcquirer(acquirerStringMap)
		if err != nil {
			return nil, errors.WithStack(err)
		}

		log.Logger.Debugf("Got acquirer %#v", acquirerImpl)

		acquirers = append(acquirers, acquirerImpl)
	}

	return acquirers, nil
}
