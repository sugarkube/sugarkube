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

package clustersot

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/sugarkube/sugarkube/internal/pkg/kapp"
	"github.com/sugarkube/sugarkube/internal/pkg/provider"
	"testing"
)

func TestNewClusterSot(t *testing.T) {
	actual, err := NewClusterSot(KUBECTL)
	assert.Nil(t, err)
	assert.Equal(t, KubeCtlClusterSot{}, actual)
}

type MockClusterSot struct {
	mock.Mock
}

func (m MockClusterSot) isOnline(sc *kapp.StackConfig, providerImpl provider.Provider) (bool, error) {
	args := m.Called(sc.Cluster)
	return args.Bool(0), args.Error(1)
}

func (m MockClusterSot) isReady(sc *kapp.StackConfig, providerImpl provider.Provider) (bool, error) {
	args := m.Called(sc.Cluster)
	return args.Bool(0), args.Error(1)
}

var testProvider = &provider.LocalProvider{}

func TestIsOnlineTrue(t *testing.T) {
	clusterName := "myCluster"

	// create an instance of our test object
	testObj := MockClusterSot{}

	// setup expectations
	testObj.On("isOnline", clusterName).Return(true, nil)

	status := kapp.ClusterStatus{IsOnline: false}
	sc := kapp.StackConfig{
		Cluster: clusterName,
		Status:  status,
	}

	assert.False(t, sc.Status.IsOnline)

	// call the code we are testing
	IsOnline(testObj, &sc, testProvider)

	// assert that the expectations were met
	testObj.AssertExpectations(t)

	assert.True(t, sc.Status.IsOnline)
}

func TestIsOnlineFalse(t *testing.T) {
	clusterName := "myCluster"

	// create an instance of our test object
	testObj := MockClusterSot{}

	// setup expectations
	testObj.On("isOnline", clusterName).Return(false, nil)

	status := kapp.ClusterStatus{IsOnline: false}
	sc := kapp.StackConfig{
		Cluster: clusterName,
		Status:  status,
	}

	assert.False(t, sc.Status.IsOnline)

	// call the code we are testing
	IsOnline(testObj, &sc, testProvider)

	// assert that the expectations were met
	testObj.AssertExpectations(t)

	assert.False(t, sc.Status.IsOnline)
}

func TestIsReadyTrue(t *testing.T) {
	clusterName := "myCluster"

	// create an instance of our test object
	testObj := MockClusterSot{}

	// setup expectations
	testObj.On("isReady", clusterName).Return(true, nil)

	status := kapp.ClusterStatus{IsReady: false}
	sc := kapp.StackConfig{
		Cluster: clusterName,
		Status:  status,
	}

	assert.False(t, sc.Status.IsReady)

	// call the code we are testing
	IsReady(testObj, &sc, testProvider)

	// assert that the expectations were met
	testObj.AssertExpectations(t)

	assert.True(t, sc.Status.IsReady)
}

func TestIsReadyFalse(t *testing.T) {
	clusterName := "myCluster"

	// create an instance of our test object
	testObj := MockClusterSot{}

	// setup expectations
	testObj.On("isReady", clusterName).Return(false, nil)

	status := kapp.ClusterStatus{IsReady: false}
	sc := kapp.StackConfig{
		Cluster: clusterName,
		Status:  status,
	}

	assert.False(t, sc.Status.IsReady)

	// call the code we are testing
	IsReady(testObj, &sc, testProvider)

	// assert that the expectations were met
	testObj.AssertExpectations(t)

	assert.False(t, sc.Status.IsReady)
}
