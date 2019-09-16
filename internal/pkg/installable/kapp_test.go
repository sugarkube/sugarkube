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
	"github.com/sugarkube/sugarkube/internal/pkg/config"
	"github.com/sugarkube/sugarkube/internal/pkg/log"
	"github.com/sugarkube/sugarkube/internal/pkg/structs"
	"os"
	"path"
	"testing"
)

const testDir = "../../testdata"

func init() {
	log.ConfigureLogger("trace", false, os.Stderr)
}

// tests adding and merging config layers
func TestAddDescriptor(t *testing.T) {
	ten := uint8(10)
	twenty := uint8(20)
	thirty := uint8(30)

	expectedKappDescriptor := structs.KappDescriptorWithMaps{
		Id: "sample-kapp",
		KappConfig: structs.KappConfig{
			Requires: []string{
				"prog2",
				"proga",
				"script",
			},
			Vars: map[string]interface{}{
				"kubeconfig": "{{ .kubeconfig }}",
				"release":    "{{ .kapp.vars.release | default .kapp.id }}",
				"helm":       "/path/to/helm",
				"region":     "{{ .stack.region }}",
			},
			RunUnits: map[string]structs.RunUnit{
				"proga": {
					WorkingDir: "/tmp",
					EnvVars: map[string]string{
						"user": "sk",
						"FOOD": "carrots",
					},
					PlanInstall: []structs.RunStep{
						{
							Name:    "print-yo",
							Command: "echo",
							Args:    "yo",
							EnvVars: map[string]string{
								"KUBECONFIG": "{{ .kapp.vars.kubeconfig }}",
							},
							MergePriority: &ten,
						},
					},
					ApplyInstall: []structs.RunStep{
						{
							Name:    "do-stuff-second",
							Command: "{{ .kapp.vars.helm }}",
							Args:    "do-stuff {{ .kapp.vars.release }}",
							EnvVars: map[string]string{
								"KUBECONFIG": "{{ .kapp.vars.kubeconfig }}",
							},
							MergePriority: &thirty,
						},
					},
				},
				"prog2": {
					Binaries: []string{"cat"},
					EnvVars:  map[string]string{},
					ApplyInstall: []structs.RunStep{
						{
							Name:    "do-stuff-first",
							Command: "/path/to/prog2",
							Args:    "do-stuff-zzz {{ .kapp.vars.region }}",
							EnvVars: map[string]string{
								"REGION": "{{ .kapp.vars.region }}",
								"COLOUR": "blue",
							},
							MergePriority: &twenty,
						},
						{
							Name:          "x",
							Command:       "/path/to/x",
							MergePriority: &ten,
						},
					},
				},
				"script": {
					EnvVars: map[string]string{},
					PlanInstall: []structs.RunStep{
						{
							Name:          "print-yes",
							Command:       "echo",
							Args:          "yes",
							MergePriority: &twenty,
						},
					},
				},
			},
		},
		Sources: map[string]structs.Source{},
		Outputs: map[string]structs.Output{},
	}

	configFile := path.Join(testDir, "test-sugarkube-conf.yaml")
	config.ViperConfig.SetConfigFile(configFile)

	err := config.Load(config.ViperConfig)
	assert.Nil(t, err)

	kapp, err := New("sample-manifest", []structs.KappDescriptorWithMaps{{Id: "sample-kapp"}})
	assert.Nil(t, err)

	err = kapp.LoadConfigFile(path.Join(testDir, "sample-workspace"))
	assert.Nil(t, err)

	descriptor := kapp.GetDescriptor()

	assert.Equal(t, expectedKappDescriptor, descriptor)
}

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
