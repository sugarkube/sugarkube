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
	"github.com/sugarkube/sugarkube/internal/pkg/constants"
	"github.com/sugarkube/sugarkube/internal/pkg/kapp"
	"github.com/sugarkube/sugarkube/internal/pkg/log"
	"strings"
	"testing"
)

func init() {
	log.ConfigureLogger("debug", false)
}

func TestCreateForward(t *testing.T) {
	// testing the correctness of stacks is handled in stack_test.go
	stackConfig, err := kapp.LoadStackConfig("standard", "../../testdata/stacks.yaml")
	assert.Nil(t, err)
	assert.NotNil(t, stackConfig)

	fakeCacheDir := "/fake/cache/dir"

	expectedPlan := Plan{
		tranches: []tranche{
			{
				manifest: *stackConfig.Manifests[0],
				tasks: []task{
					{
						action: constants.TaskActionInstall,
						kapp:   stackConfig.Manifests[0].ParsedKapps()[0],
					},
				},
			},
			{
				manifest: *stackConfig.Manifests[1],
				tasks: []task{
					{
						action: constants.TaskActionInstall,
						kapp:   stackConfig.Manifests[1].ParsedKapps()[0],
					},
					{
						action: constants.TaskActionInstall,
						kapp:   stackConfig.Manifests[1].ParsedKapps()[1],
					},
					{
						action: constants.TaskActionInstall,
						kapp:   stackConfig.Manifests[1].ParsedKapps()[2],
					},
					{
						action: constants.TaskActionInstall,
						kapp:   stackConfig.Manifests[1].ParsedKapps()[3],
					},
				},
			},
			{ // manifest3.yaml
				manifest: *stackConfig.Manifests[2],
				tasks: []task{
					{
						action: constants.TaskActionInstall,
						kapp:   stackConfig.Manifests[2].ParsedKapps()[0],
					},
				},
			},
			{ // manifest3.yaml - this kapp has an additional tranche for the cluster update post action
				manifest: *stackConfig.Manifests[2],
				tasks: []task{
					{
						action: constants.TaskActionClusterUpdate,
						kapp:   stackConfig.Manifests[2].ParsedKapps()[0],
					},
				},
			},
			{ // manifest3.yaml
				manifest: *stackConfig.Manifests[2],
				tasks: []task{
					{
						action: constants.TaskActionInstall,
						kapp:   stackConfig.Manifests[2].ParsedKapps()[1],
					},
				},
			},
			{
				manifest: *stackConfig.Manifests[3],
				tasks: []task{
					{
						action: constants.TaskActionDestroy,
						kapp:   stackConfig.Manifests[3].ParsedKapps()[0],
					},
					{
						action: constants.TaskActionInstall,
						kapp:   stackConfig.Manifests[3].ParsedKapps()[1],
					},
				},
			},
		},
		stackConfig:     stackConfig,
		cacheDir:        fakeCacheDir,
		renderTemplates: true,
	}

	actionPlan, err := Create(true, stackConfig, stackConfig.Manifests,
		fakeCacheDir, []string{}, []string{}, true)
	assert.Nil(t, err)

	// assert that manifests are in the correct order in stack configs
	assert.True(t, strings.HasSuffix(stackConfig.Manifests[0].Uri, "manifest1.yaml"))
	assert.True(t, strings.HasSuffix(stackConfig.Manifests[1].Uri, "manifest2.yaml"))
	assert.True(t, strings.HasSuffix(stackConfig.Manifests[2].Uri, "manifest3.yaml"))
	assert.True(t, strings.HasSuffix(stackConfig.Manifests[3].Uri, "manifest4.yaml"))

	assert.Equal(t, expectedPlan.tranches[0].manifest.Uri, actionPlan.tranches[0].manifest.Uri)

	assert.Equal(t, expectedPlan, *actionPlan)
}

func TestCreateReverse(t *testing.T) {
	// testing the correctness of stacks is handled in stack_test.go
	stackConfig, err := kapp.LoadStackConfig("standard", "../../testdata/stacks.yaml")
	assert.Nil(t, err)
	assert.NotNil(t, stackConfig)

	fakeCacheDir := "/fake/cache/dir"

	expectedPlan := Plan{
		tranches: []tranche{
			{
				manifest: *stackConfig.Manifests[3],
				tasks: []task{
					{
						action: constants.TaskActionDestroy,
						kapp:   stackConfig.Manifests[3].ParsedKapps()[1],
					},
					{
						action: constants.TaskActionDestroy,
						kapp:   stackConfig.Manifests[3].ParsedKapps()[0],
					},
				},
			},
			{ // manifest3.yaml
				manifest: *stackConfig.Manifests[2],
				tasks: []task{
					{
						action: constants.TaskActionDestroy,
						kapp:   stackConfig.Manifests[2].ParsedKapps()[1],
					},
				},
			},
			{ // manifest3.yaml - this kapp has an additional tranche for the cluster update post action
				manifest: *stackConfig.Manifests[2],
				tasks: []task{
					{
						action: constants.TaskActionClusterUpdate,
						kapp:   stackConfig.Manifests[2].ParsedKapps()[0],
					},
				},
			},
			{ // manifest3.yaml
				manifest: *stackConfig.Manifests[2],
				tasks: []task{
					{
						action: constants.TaskActionDestroy,
						kapp:   stackConfig.Manifests[2].ParsedKapps()[0],
					},
				},
			},
			{
				manifest: *stackConfig.Manifests[1],
				tasks: []task{
					{
						action: constants.TaskActionDestroy,
						kapp:   stackConfig.Manifests[1].ParsedKapps()[3],
					},
					{
						action: constants.TaskActionDestroy,
						kapp:   stackConfig.Manifests[1].ParsedKapps()[2],
					},
					{
						action: constants.TaskActionDestroy,
						kapp:   stackConfig.Manifests[1].ParsedKapps()[1],
					},
					{
						action: constants.TaskActionDestroy,
						kapp:   stackConfig.Manifests[1].ParsedKapps()[0],
					},
				},
			},
			{
				manifest: *stackConfig.Manifests[0],
				tasks: []task{
					{
						action: constants.TaskActionDestroy,
						kapp:   stackConfig.Manifests[0].ParsedKapps()[0],
					},
				},
			},
		},
		stackConfig:     stackConfig,
		cacheDir:        fakeCacheDir,
		renderTemplates: true,
	}

	actionPlan, err := Create(false, stackConfig, stackConfig.Manifests,
		fakeCacheDir, []string{}, []string{}, true)
	assert.Nil(t, err)

	// assert that manifests are in the correct order in stack configs
	assert.True(t, strings.HasSuffix(stackConfig.Manifests[0].Uri, "manifest1.yaml"))
	assert.True(t, strings.HasSuffix(stackConfig.Manifests[1].Uri, "manifest2.yaml"))
	assert.True(t, strings.HasSuffix(stackConfig.Manifests[2].Uri, "manifest3.yaml"))
	assert.True(t, strings.HasSuffix(stackConfig.Manifests[3].Uri, "manifest4.yaml"))

	assert.Equal(t, expectedPlan.tranches[0].manifest.Uri, actionPlan.tranches[0].manifest.Uri)

	assert.Equal(t, expectedPlan, *actionPlan)
}
