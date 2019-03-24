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

package provisioner

import (
	"github.com/stretchr/testify/assert"
	"github.com/sugarkube/sugarkube/internal/pkg/log"
	"testing"
)

func init() {
	log.ConfigureLogger("debug", false)
}

func TestNewNonExistentProvisioner(t *testing.T) {
	actual, err := New("bananas", nil, nil)
	assert.NotNil(t, err)
	assert.Nil(t, actual)
}

//func TestNewMinikubeProvisioner(t *testing.T) {
//
//	stackObj, err := stack.BuildStack("standard", "../../testdata/stacks.yaml",
//		&structs.Stack{}, &config.Config{}, os.Stdout)
//	assert.Nil(t, err)
//
//	clusterSot, err := clustersot.New(clustersot.KubeCtl, stackObj)
//	assert.Nil(t, err)
//
//	actual, err := New(MinikubeProvisionerName, stackObj, clusterSot)
//	assert.Nil(t, err)
//	assert.Equal(t, MinikubeProvisioner{
//		clusterSot: clusterSot,
//		stack:      stackObj,
//		minikubeConfig: MinikubeConfig{
//			Binary: "minikube",
//			Params: struct {
//				Global map[string]string
//				Start  map[string]string
//			}{
//				nil,
//				map[string]string{
//					"cpus":      "2",
//					"disk_size": "30g",
//					"memory":    "2048",
//					"should_be": "present",
//				},
//			},
//		},
//	}, actual)
//}
//
//func TestNewKopsProvisioner(t *testing.T) {
//	stackObj, err := stack.BuildStack("standard", "../../testdata/stacks.yaml",
//		&structs.Stack{}, &config.Config{}, os.Stdout)
//	assert.Nil(t, err)
//
//	clusterSot, err := clustersot.New(clustersot.KubeCtl, stackObj)
//	assert.Nil(t, err)
//
//	actual, err := New(kopsProvisionerName, stackObj, clusterSot)
//	assert.Nil(t, err)
//	assert.Equal(t, KopsProvisioner{
//		stack:      stackObj,
//		clusterSot: clusterSot,
//		kopsConfig: KopsConfig{
//			Binary: "kops",
//		},
//	}, actual)
//}
//
//func TestNewNoOpProvisioner(t *testing.T) {
//	stackObj, err := stack.BuildStack("standard", "../../testdata/stacks.yaml",
//		&structs.Stack{}, &config.Config{}, os.Stdout)
//	assert.Nil(t, err)
//
//	clusterSot, err := clustersot.New(clustersot.KubeCtl, stackObj)
//	assert.Nil(t, err)
//
//	actual, err := New(NoopProvisionerName, stackObj, clusterSot)
//	assert.Nil(t, err)
//	assert.Equal(t, NoOpProvisioner{}, actual)
//}
