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

package plan

import (
	"github.com/stretchr/testify/assert"
	"github.com/sugarkube/sugarkube/internal/pkg/interfaces"
	"github.com/sugarkube/sugarkube/internal/pkg/log"
	"github.com/sugarkube/sugarkube/internal/pkg/stack"
	"github.com/sugarkube/sugarkube/internal/pkg/structs"
	"path/filepath"
	"testing"
)

func init() {
	log.ConfigureLogger("debug", false)
}

const testDir = "../../testdata"

func TestFindDependencies(t *testing.T) {
	absTestDir, err := filepath.Abs(testDir)
	assert.Nil(t, err)

	manifestPaths := []string{
		// has parallelisation == 1
		filepath.Join(absTestDir, "manifests/manifest2.yaml"),
		// doesn't set the parallelisation
		filepath.Join(absTestDir, "manifests/manifest3.yaml"),
	}

	manifests := make([]interfaces.IManifest, 0)

	for _, manifestPath := range manifestPaths {
		manifestDescriptor := structs.ManifestDescriptor{
			Uri: manifestPath,
		}
		manifest, err := stack.ParseManifestFile(manifestPath, manifestDescriptor)
		assert.Nil(t, err)
		manifests = append(manifests, manifest)
	}

	descriptors := findDependencies(manifests)

	expected := map[string]nodeDescriptor{
		"manifest2:kappC": {dependsOn: []string{}, installableObj: manifests[0].Installables()[0]},
		"manifest2:kappB": {dependsOn: []string{"manifest2:kappC"}, installableObj: manifests[0].Installables()[1]},
		"manifest2:kappD": {dependsOn: []string{"manifest2:kappB"}, installableObj: manifests[0].Installables()[2]},
		"manifest2:kappA": {dependsOn: []string{"manifest2:kappD"}, installableObj: manifests[0].Installables()[3]},
		"manifest3:kappX": {dependsOn: []string{}, installableObj: manifests[1].Installables()[0]},
		"manifest3:kappY": {dependsOn: []string{}, installableObj: manifests[1].Installables()[1]},
	}

	assert.Equal(t, expected, descriptors)
}
