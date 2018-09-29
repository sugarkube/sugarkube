// +build integration

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
	"io/ioutil"
	"os"
	"testing"
)

func TestGitAcquire(t *testing.T) {
	acquirer, err := NewAcquirer(defaultSettings)
	assert.Nil(t, err)

	tempDir, err := ioutil.TempDir("", "git-")
	assert.Nil(t, err)

	log.Logger.Debugf("Testing the git acquirer with temp dir: %s", tempDir)
	defer os.RemoveAll(tempDir)

	err = acquirer.acquire(tempDir)
	assert.Nil(t, err)
}
