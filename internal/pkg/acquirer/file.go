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

package acquirer

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/sugarkube/sugarkube/internal/pkg/log"
	"os"
	"path/filepath"
	"strings"
)

const FILE_ACQUIRER = "file"
const FILE_PROTOCOL = "file://"

// An acquirer for files already on the local filesystem. This is analogous to a no-op acquirer and allows us
// to use a common interface for acquisition operations
type FileAcquirer struct {
	name string
	uri  string
}

// Returns an instance. This allows us to build objects for testing instead of
// directly instantiating objects in the acquirer factory.
func NewFileAcquirer(name string, uri string) (*FileAcquirer, error) {

	if strings.TrimSpace(uri) == "" {
		return nil, errors.New("Missing URI for file acquirer")
	}

	if !strings.HasPrefix(uri, FILE_PROTOCOL) {
		return nil, errors.New(
			fmt.Sprintf("Unexpected file URI. Expected a single ':' "+
				"character in URI %s", uri))
	}

	if strings.TrimSpace(name) == "" {
		name = filepath.Base(uri)
	}

	return &FileAcquirer{
		name: name,
		uri:  strings.TrimSpace(uri),
	}, nil
}

// Returns an ID, either the explicitly set name or the basename of the last component
func (a FileAcquirer) Id() (string, error) {
	return filepath.Base(a.Path()), nil
}

// return the name
func (a FileAcquirer) Name() string {
	return a.name
}

// return the path (i.e. the URI with the file:// prefix removed)
func (a FileAcquirer) Path() string {
	return strings.TrimPrefix(a.uri, FILE_PROTOCOL)
}

// return whether this source should be searched for values files
func (a FileAcquirer) IncludeValues() bool {
	// todo - delete this method if we don't need it
	return false
}

// Verifies the file already exists and returns an error if not
func (a FileAcquirer) acquire(dest string) error {

	if _, err := os.Stat(dest); err != nil {
		if os.IsNotExist(err) {
			log.Logger.Errorf("File '%s' doesn't exist", dest)
		} else {
			return errors.WithStack(err)
		}
	} else {
		log.Logger.Debugf("File '%s' already exists as expected", dest)
	}

	return nil
}
