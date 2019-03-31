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
	"github.com/sugarkube/sugarkube/internal/pkg/log"
)

func init() {
	log.ConfigureLogger("debug", false)
}

// todo - test adding and merging config layers

//func TestFindKappVarsFiles(t *testing.T) {
//
//	absTestDir, err := filepath.Abs(testDir)
//	assert.Nil(t, err)
//
//	manifest1, manifest2 := GetTestManifests()
//
//	stackConfig := structs.Stack{
//		Name:        "large",
//		FilePath:    "../../testdata/stacks.yaml",
//		Provider:    "test-provider",
//		Provisioner: "test-provisioner",
//		Profile:     "test-profile",
//		Cluster:     "test-cluster",
//		Account:     "test-account",
//		Region:      "test-region1",
//		ProviderVarsDirs: []string{
//			"./stacks/",
//		},
//		KappVarsDirs: []string{
//			"./sample-kapp-vars/kapp-vars/",
//			"./sample-kapp-vars/kapp-vars2/",
//		},
//		Manifests: []*Manifest{
//			manifest1,
//			manifest2,
//		},
//	}
//
//	expected := []string{
//		filepath.Join(absTestDir, "sample-kapp-vars/kapp-vars/test-provider/test-provisioner/test-profile.yaml"),
//		filepath.Join(absTestDir, "sample-kapp-vars/kapp-vars/test-provider/test-provisioner/test-account/values.yaml"),
//		filepath.Join(absTestDir, "sample-kapp-vars/kapp-vars/test-provider/test-provisioner/test-account/test-region1/kappA.yaml"),
//		filepath.Join(absTestDir, "sample-kapp-vars/kapp-vars/test-provider/test-provisioner/test-account/test-region1/values.yaml"),
//		filepath.Join(absTestDir, "sample-kapp-vars/kapp-vars2/kappA.yaml"),
//	}
//
//	kappObj := stackObj.GetConfig().Manifests()[0].Installables()[0]
//	results, err := kappObj.(*Kapp).findVarsFiles(stackObj.GetConfig())
//	assert.Nil(t, err)
//
//	assert.Equal(t, expected, results)
//}
//
//func TestMergeVarsForKapp(t *testing.T) {
//
//	// testing the correctness of this stack is handled in stack_test.go
//	stackConfig, err := stack.BuildStack("large", "../../testdata/stacks.yaml",
//		&structs.StackFile{}, &config.Config{}, os.Stdout)
//	assert.Nil(t, err)
//	assert.NotNil(t, stackConfig)
//
//	expectedVarsFromFiles := map[string]interface{}{
//		"colours": []interface{}{
//			"green",
//		},
//		"location": "kappFile",
//	}
//
//	kappObj := stackConfig.GetConfig().Manifests()[0].Installables()[0]
//
//	results, err := kappObj.Vars(stackConfig)
//	assert.Nil(t, err)
//
//	assert.Equal(t, expectedVarsFromFiles, results)
//
//	// now we've loaded kapp variables from a file, test merging vars for the kapp
//	expectedMergedVars := map[string]interface{}{
//		"stack": map[interface{}]interface{}{
//			"name":        "large",
//			"profile":     "local",
//			"provider":    "local",
//			"provisioner": "minikube",
//			"region":      "",
//			"account":     "",
//			"cluster":     "large",
//			"filePath":    "../../testdata/stacks.yaml",
//		},
//		"sugarkube": map[interface{}]interface{}{
//			"target":   "myTarget",
//			"approved": true,
//			"defaultVars": []interface{}{
//				"local",
//				"",
//				"local",
//				"large",
//				"",
//			},
//		},
//		"kapp": map[interface{}]interface{}{
//			"id":        "kappA",
//			"state":     "absent",
//			"cacheRoot": "manifest1/kappA",
//			"vars": map[interface{}]interface{}{
//				"colours": []interface{}{
//					"red",
//					"black",
//				},
//				"location": "kappFile",
//				"sizeVar":  "mediumOverridden",
//				"stackVar": "setInOverrides",
//			},
//		},
//	}
//
//	stackObj, err := stack.newStack(stackConfig, nil, nil)
//	assert.Nil(t, err)
//
//	templatedVars, err := stackObj.GetTemplatedVars(kappObj,
//		map[string]interface{}{"target": "myTarget", "approved": true})
//
//	assert.Equal(t, expectedMergedVars, templatedVars)
//}
