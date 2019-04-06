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
	"github.com/sugarkube/sugarkube/internal/pkg/structs"
	"strings"
)

type Acquirer interface {
	acquire(dest string) error
	FullyQualifiedId() (string, error)
	Id() string
	Path() string
	Uri() string
}

// Instantiates a new acquirer from a source
func New(source structs.Source) (Acquirer, error) {

	if strings.Contains(source.Uri, ".git") {
		acquirerObj, err := newGitAcquirer(source)
		if err != nil {
			return nil, errors.WithStack(err)
		}
		return acquirerObj, nil
	} else if strings.HasPrefix(source.Uri, FileProtocol) {

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

// Takes a list of Sources and returns a list of instantiated acquirers that represent them
func GetAcquirersFromSources(sources map[string]structs.Source) (map[string]Acquirer, error) {
	acquirers := make(map[string]Acquirer, len(sources))

	for key, source := range sources {
		acquirer, err := New(source)
		if err != nil {
			return nil, errors.WithStack(err)
		}
		acquirers[key] = acquirer
	}

	return acquirers, nil
}
