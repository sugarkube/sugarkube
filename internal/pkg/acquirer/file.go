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
	"github.com/sugarkube/sugarkube/internal/pkg/structs"
	"os"
	"path/filepath"
	"strings"
)

const FileProtocol = "file://"

// An acquirer for files already on the local filesystem. This is analogous to a no-op acquirer and allows us
// to use a common interface for acquisition operations
type FileAcquirer struct {
	id  string
	uri string
}

// Returns an instance. This allows us to build objects for testing instead of
// directly instantiating objects in the acquirer factory.
func newFileAcquirer(source structs.Source) (*FileAcquirer, error) {

	uri := source.Uri

	if uri == "" {
		return nil, errors.New("Missing URI for file acquirer")
	}

	if !strings.HasPrefix(uri, FileProtocol) {
		return nil, errors.New(
			fmt.Sprintf("Unexpected file URI. Expected a single ':' "+
				"character in URI %s", uri))
	}

	id := source.Id

	if id == "" {
		id = filepath.Base(uri)
	}

	return &FileAcquirer{
		id:  id,
		uri: strings.TrimSpace(uri),
	}, nil
}

// Returns an ID, either the explicitly set id or the basename of the last component
func (a FileAcquirer) FullyQualifiedId() (string, error) {
	return filepath.Base(a.Path()), nil
}

// return the id
func (a FileAcquirer) Id() string {
	return a.id
}

// return the path (i.e. the URI with the file:// prefix removed)
func (a FileAcquirer) Path() string {
	return strings.TrimPrefix(a.uri, FileProtocol)
}

// return the path (i.e. the URI with the file:// prefix removed)
func (a FileAcquirer) Uri() string {
	return a.uri
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
