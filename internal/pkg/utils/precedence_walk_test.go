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

package utils

import (
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/sugarkube/sugarkube/internal/pkg/log"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"
)

func init() {
	log.ConfigureLogger("debug", false, os.Stderr)
}

func TestPrecedenceWalk(t *testing.T) {
	absTestDir, err := filepath.Abs(testDir)
	assert.Nil(t, err)

	precedence := []string{
		"values",
		"aws",
		"test-account",
		"test-profile",
		"test-cluster",
		"region1",
		"accounts",
		"profiles",
		"clusters",
	}

	expected := []string{
		"providers/values.yaml",
		"providers/region1.yaml",
		"providers/aws/accounts/test-account/values.yaml",
		"providers/aws/accounts/test-account/region1.yaml",
		"providers/aws/accounts/test-account/profiles/test-profile/clusters/test-cluster/values.yaml",
		"providers/aws/accounts/test-account/profiles/test-profile/clusters/test-cluster/region1/values.yaml",
		"providers/test-account/region1.yaml",
		"providers/test-account/test-cluster/values.yaml",
		"providers/region1/values.yaml",
		"providers/region1/test-cluster.yaml",
	}

	visited := make([]string, 0)

	startDir := path.Join(absTestDir, "providers")

	err = PrecedenceWalk(startDir, precedence, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return errors.WithStack(err)
		}

		if !info.IsDir() {
			path = strings.TrimPrefix(path, absTestDir)
			path = strings.TrimPrefix(path, "/")
			visited = append(visited, path)

			log.Logger.Debugf("Walked to file: %s", path)
		}
		return nil
	})

	assert.Nil(t, err)
	assert.Equal(t, expected, visited)
}

func TestPrecedenceWalkDedupe(t *testing.T) {
	absTestDir, err := filepath.Abs(testDir)
	assert.Nil(t, err)

	precedence := []string{
		"values",
		"aws",
		"test-account",
		"test-profile",
		"test-cluster",
		"test-profile", // sometimes we have the same value for account/profile/cluster, etc.
		"region1",
		"test-profile",
		"accounts",
		"profiles",
		"clusters",
	}

	expected := []string{
		"providers/values.yaml",
		"providers/region1.yaml",
		"providers/aws/accounts/test-account/values.yaml",
		"providers/aws/accounts/test-account/region1.yaml",
		"providers/aws/accounts/test-account/profiles/test-profile/clusters/test-cluster/values.yaml",
		"providers/aws/accounts/test-account/profiles/test-profile/clusters/test-cluster/region1/values.yaml",
		"providers/test-account/region1.yaml",
		"providers/test-account/test-cluster/values.yaml",
		"providers/region1/values.yaml",
		"providers/region1/test-cluster.yaml",
	}

	visited := make([]string, 0)

	startDir := path.Join(absTestDir, "providers")

	err = PrecedenceWalk(startDir, precedence, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return errors.WithStack(err)
		}

		if !info.IsDir() {
			path = strings.TrimPrefix(path, absTestDir)
			path = strings.TrimPrefix(path, "/")
			visited = append(visited, path)

			log.Logger.Debugf("Walked to file: %s", path)
		}
		return nil
	})

	assert.Nil(t, err)
	assert.Equal(t, expected, visited)
}
