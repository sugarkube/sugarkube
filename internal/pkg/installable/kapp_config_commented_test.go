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

package installable

import (
	"github.com/stretchr/testify/assert"
	"github.com/sugarkube/sugarkube/internal/pkg/log"
	"github.com/sugarkube/sugarkube/internal/pkg/structs"
	"path"
	"testing"
)

const testDir = "../../testdata"

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
		"make": {
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

	testKapp := Kapp{
		descriptor: structs.KappDescriptor{
			Id: "sample-kapp",
		},
		manifestId: "sample-manifest",
	}
	testKapp.SetRootCacheDir(path.Join(testDir, "sample-cache"))

	err := testKapp.RefreshConfig(templateVars)
	assert.Nil(t, err)

	assert.Equal(t, expectedEnvVars, testKapp.config.EnvVars)
	assert.Equal(t, []string{"helm"}, testKapp.config.Requires)
	assert.Equal(t, expectedArgs, testKapp.config.Args)
}

//func TestMergeProgramConfigs(t *testing.T) {
//	kappId := "sample-kapp"
//	stackRegion := "testRegion"
//	testContext := "test-context"
//
//	templateVars := map[string]interface{}{
//		"kube_context": testContext,
//		"kapp": map[string]interface{}{
//			"id": kappId,
//		},
//		"stack": map[string]interface{}{
//			"region": stackRegion,
//		},
//	}
//
//	testKapp := Kapp{Id: "sample-kapp",
//		manifest: &Manifest{
//			ConfiguredId: "sample-manifest",
//		},
//	}
//	testKapp.SetRootCacheDir(path.Join(testDir, "sample-cache"))
//
//	// load the kapp
//	err := testKapp.Load(templateVars)
//	assert.Nil(t, err)
//
//	// load the test config file - it's tested elsewhere
//	configFile := path.Join(testDir, "test-sugarkube-conf.yaml")
//	config.ViperConfig.SetConfigFile(configFile)
//	err = config.Load(config.ViperConfig)
//	assert.Nil(t, err)
//
//	// load the test stack config
//	stackConfig, err := LoadStackConfig("large", "../../testdata/stacks.yaml")
//	assert.Nil(t, err)
//
//	// create a stack
//	stackObj := &structs.Stack{
//		Config:      stackConfig,
//		Provider:    nil,
//		Provisioner: nil,
//		Status: nil,
//	}
//
//	mergedVars, err := stackObj.TemplatedVars(nil, map[string]interface{}{})
//	assert.Nil(t, err)
//
//	// get the merged config for the kapp
//	mergedConfig, err := testKapp.MergeProgramConfigs(config.CurrentConfig.Programs,
//		mergedVars)
//	assert.Nil(t, err)
//
//	log.Logger.Fatalf("mergedconfig=%#v", mergedConfig)
//}
