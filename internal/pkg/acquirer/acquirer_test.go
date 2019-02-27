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
	"testing"
)

const GOOD_GIT_URI = "git@github.com:sugarkube/kapps.git//incubator/tiller/#master"

func init() {
	log.ConfigureLogger("debug", false)
}

func TestNewAcquirerError(t *testing.T) {
	actual, err := newAcquirer(Source{Id: "nonsense"})
	assert.NotNil(t, err)
	assert.Nil(t, actual)
}

func TestNewAcquirerFile(t *testing.T) {
	var expectedAcquirer = &FileAcquirer{
		id:  "test.txt",
		uri: "file:///tmp/test.txt",
	}

	actual, err := newAcquirer(Source{Uri: "file:///tmp/test.txt"})
	assert.Nil(t, err)
	assert.Equal(t, expectedAcquirer, actual)
	assert.Equal(t, "/tmp/test.txt", actual.Path())
}

func TestNewAcquirerGit(t *testing.T) {
	var expectedAcquirer = &GitAcquirer{
		id:            "tiller",
		uri:           "git@github.com:sugarkube/kapps.git",
		branch:        "master",
		path:          "incubator/tiller/",
		includeValues: true,
	}

	actual, err := newAcquirer(Source{
		Id:            "",
		Uri:           GOOD_GIT_URI,
		IncludeValues: true,
	})
	assert.Nil(t, err)
	assert.Equal(t, expectedAcquirer, actual,
		"Fully-defined git acquirer incorrectly created")
}

func TestNewAcquirerGitNoBranch(t *testing.T) {

	actual, err := newAcquirer(Source{
		Id:            "",
		Uri:           "git@github.com:sugarkube/kapps.git//incubator/tiller/",
		IncludeValues: true,
	})
	assert.NotNil(t, err)
	assert.Nil(t, actual)
}

func TestNewAcquirerGitWithOptions(t *testing.T) {
	var expectedAcquirer = &GitAcquirer{
		id:            "tiller",
		uri:           "git@github.com:sugarkube/kapps.git",
		branch:        "my-branch",
		path:          "incubator/tiller/",
		includeValues: true,
	}

	actual, err := newAcquirer(Source{
		Id:  "",
		Uri: GOOD_GIT_URI,
		Options: map[string]interface{}{
			"branch": "my-branch",
		},
		IncludeValues: true,
	})
	assert.Nil(t, err)
	assert.Equal(t, expectedAcquirer, actual,
		"Git acquirer with additional options incorrectly created")
}

func TestNewAcquirerGitWithOptionsNoDefault(t *testing.T) {
	var expectedAcquirer = &GitAcquirer{
		id:            "tiller",
		uri:           "git@github.com:sugarkube/kapps.git",
		branch:        "my-branch",
		path:          "incubator/tiller/",
		includeValues: true,
	}

	actual, err := newAcquirer(Source{
		Id: "",
		// this URI has no branch at all so it must come from the options
		Uri: "git@github.com:sugarkube/kapps.git//incubator/tiller/",
		Options: map[string]interface{}{
			"branch": "my-branch",
		},
		IncludeValues: true,
	})
	assert.Nil(t, err)
	assert.Equal(t, expectedAcquirer, actual,
		"Git acquirer with additional options incorrectly created")
}

func TestNewAcquirerGitExplicitId(t *testing.T) {
	var expectedAcquirer = &GitAcquirer{
		id:            "banana",
		uri:           "git@github.com:sugarkube/kapps.git",
		branch:        "master",
		path:          "incubator/tiller/",
		includeValues: false,
	}

	actual, err := newAcquirer(Source{Id: "banana", Uri: GOOD_GIT_URI})
	assert.Nil(t, err)
	assert.Equal(t, expectedAcquirer, actual,
		"Git acquirer with explicitly set ID incorrectly created")
}
