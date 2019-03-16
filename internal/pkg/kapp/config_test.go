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

package kapp

import (
	"github.com/stretchr/testify/assert"
	"github.com/sugarkube/sugarkube/internal/pkg/log"
	"path"
	"testing"
)

func init() {
	log.ConfigureLogger("debug", false)
}

func TestLoad(t *testing.T) {
	kappId := "sample-kapp"
	stackRegion := "testRegion"
	testContext := "test-context"

	expectedEnvVars := map[string]interface{}{
		"STATIC":       "someValue",
		"KUBE_CONTEXT": testContext,
		"NAMESPACE":    kappId,
		"REGION":       stackRegion,
	}

	expectedArgs := map[string]map[string][]map[string]string{
		"targets": {
			"install": {
				{
					"name":  "helm-opts",
					"value": "yes",
				},
			},
		},
	}

	templateVars := map[string]interface{}{
		"kube_context": testContext,
		"kapp": map[string]interface{}{
			"id": kappId,
		},
		"stack": map[string]interface{}{
			"region": stackRegion,
		},
	}

	testKapp := Kapp{Id: "sample-kapp",
		manifest: &Manifest{
			ConfiguredId: "sample-manifest",
		},
	}
	testKapp.SetCacheDir(path.Join(testDir, "sample-cache"))

	err := testKapp.Load(templateVars)
	assert.Nil(t, err)

	assert.Equal(t, expectedEnvVars, testKapp.Config.EnvVars)
	assert.Equal(t, []string{"helm"}, testKapp.Config.Requires)
	assert.Equal(t, expectedArgs, testKapp.Config.Args)
}
