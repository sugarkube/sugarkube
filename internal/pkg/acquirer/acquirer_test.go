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
	"github.com/stretchr/testify/assert"
	"github.com/sugarkube/sugarkube/internal/pkg/log"
	"github.com/sugarkube/sugarkube/internal/pkg/structs"
	"os"
	"testing"
)

const GoodGitUri = "git@github.com:sugarkube/kapps.git//incubator/tiller#master"

func init() {
	log.ConfigureLogger("debug", false, os.Stderr)
}

func TestNewAcquirerError(t *testing.T) {
	actual, err := New(structs.Source{Id: "nonsense"}, "test-id", true)
	assert.NotNil(t, err)
	assert.Nil(t, actual)
}

func TestNewAcquirerFile(t *testing.T) {
	var expectedAcquirer = &FileAcquirer{
		id:  "test.txt",
		uri: "file:///tmp/test.txt",
	}

	actual, err := New(structs.Source{Uri: "file:///tmp/test.txt"}, "test-id", true)
	assert.Nil(t, err)
	assert.Equal(t, expectedAcquirer, actual)
	assert.Equal(t, "/tmp/test.txt", actual.Path())
}

func TestNewAcquirerGit(t *testing.T) {
	var expectedAcquirer = &GitAcquirer{
		id:     "tiller",
		uri:    "git@github.com:sugarkube/kapps.git",
		branch: "master",
		path:   "incubator/tiller",
	}

	actual, err := New(structs.Source{
		Id:  "",
		Uri: GoodGitUri,
	}, "test-id", true)
	assert.Nil(t, err)
	assert.Equal(t, expectedAcquirer, actual,
		"Fully-defined git acquirer incorrectly created")
}
func TestNewAcquirerGitHttps(t *testing.T) {
	var expectedAcquirer = &GitAcquirer{
		id:     "tiller",
		uri:    "https://github.com/sugarkube/sugarkube.git",
		branch: "master",
		path:   "incubator/tiller",
	}

	actual, err := New(structs.Source{
		Id:  "",
		Uri: "https://github.com/sugarkube/sugarkube.git//incubator/tiller#master",
	}, "test-id", true)
	assert.Nil(t, err)
	assert.Equal(t, expectedAcquirer, actual,
		"Fully-defined HTTPS git acquirer incorrectly created")
}

func TestNewAcquirerGitNoBranch(t *testing.T) {

	actual, err := New(structs.Source{
		Id:  "",
		Uri: "git@github.com:sugarkube/kapps.git//incubator/tiller/",
	}, "test-id", true)
	assert.NotNil(t, err)
	assert.Nil(t, actual)
}

func TestNewAcquirerGitWithOptions(t *testing.T) {
	var expectedAcquirer = &GitAcquirer{
		id:     "tiller",
		uri:    "git@github.com:sugarkube/kapps.git",
		branch: "my-branch",
		path:   "incubator/tiller",
	}

	actual, err := New(structs.Source{
		Id:  "",
		Uri: GoodGitUri,
		Options: map[string]interface{}{
			"branch": "my-branch",
		},
	}, "test-id", true)
	assert.Nil(t, err)
	assert.Equal(t, expectedAcquirer, actual,
		"Git acquirer with additional options incorrectly created")
}

func TestNewAcquirerGitWithOptionsNoDefault(t *testing.T) {
	var expectedAcquirer = &GitAcquirer{
		id:     "tiller",
		uri:    "git@github.com:sugarkube/kapps.git",
		branch: "my-branch",
		path:   "incubator/tiller/",
	}

	actual, err := New(structs.Source{
		Id: "",
		// this URI has no branch at all so it must come from the options
		Uri: "git@github.com:sugarkube/kapps.git//incubator/tiller/",
		Options: map[string]interface{}{
			"branch": "my-branch",
		},
	}, "test-id", true)
	assert.Nil(t, err)
	assert.Equal(t, expectedAcquirer, actual,
		"Git acquirer with additional options incorrectly created")
}

func TestNewAcquirerGitExplicitId(t *testing.T) {
	var expectedAcquirer = &GitAcquirer{
		id:     "banana",
		uri:    "git@github.com:sugarkube/kapps.git",
		branch: "master",
		path:   "incubator/tiller",
	}

	actual, err := New(structs.Source{Id: "banana", Uri: GoodGitUri}, "test-id", true)
	assert.Nil(t, err)
	assert.Equal(t, expectedAcquirer, actual,
		"Git acquirer with explicitly set ID incorrectly created")
}
