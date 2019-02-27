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
	"strings"
)

type Acquirer interface {
	acquire(dest string) error
	FullyQualifiedId() (string, error)
	Id() string
	Path() string
	Uri() string
	IncludeValues() bool // todo - clarify if this is actually used, and if not, remove it
}

type Source struct {
	Id  string
	Uri string
	// todo - implement
	Options       map[string]interface{}
	IncludeValues bool // todo - see if we actually need this
}

// Instantiates a new acquirer from a source
func newAcquirer(source Source) (Acquirer, error) {

	if strings.Contains(source.Uri, ".git") {
		acquirerObj, err := NewGitAcquirer(source)
		if err != nil {
			return nil, errors.WithStack(err)
		}
		return acquirerObj, nil
	} else if strings.HasPrefix(source.Uri, FILE_PROTOCOL) {

		acquirerObj, err := newFileAcquirer(source)
		if err != nil {
			return nil, errors.WithStack(err)
		}
		return acquirerObj, nil
	}

	return nil, errors.New(fmt.Sprintf("Couldn't identify acquirer for URI '%s'", source.Uri))
}

// Delegate to an acquirer implementation
func Acquire(a Acquirer, dest string) error {
	return a.acquire(dest)
}

// Parse acquirers from a values map
//func ParseAcquirers(acquirerMaps []map[interface{}]interface{}) ([]Acquirer, error) {
//	acquirers := make([]Acquirer, 0)
//	// now we have a list of sources, get the acquirer for each one
//	for _, acquirerMap := range acquirerMaps {
//		acquirerStringMap, err := convert.MapInterfaceInterfaceToMapStringString(acquirerMap)
//		if err != nil {
//			return nil, errors.WithStack(err)
//		}
//
//		acquirerImpl, err := NewAcquirer(acquirerStringMap)
//		if err != nil {
//			return nil, errors.WithStack(err)
//		}
//
//		log.Logger.Debugf("Got acquirer %#v", acquirerImpl)
//
//		acquirers = append(acquirers, acquirerImpl)
//	}
//
//	return acquirers, nil
//}

// Takes a list of Sources and returns a list of instantiated acquirers that represent them
func GetAcquirersFromSources(sources []Source) ([]Acquirer, error) {
	acquirers := make([]Acquirer, len(sources))

	for i, source := range sources {
		acquirer, err := newAcquirer(source)
		if err != nil {
			return nil, errors.WithStack(err)
		}
		acquirers[i] = acquirer
	}

	return acquirers, nil
}
