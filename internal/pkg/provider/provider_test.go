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

package provider

import (
	"github.com/stretchr/testify/assert"
	"github.com/sugarkube/sugarkube/internal/pkg/kapp"
	"testing"
)

func TestStackConfigVars(t *testing.T) {
	sc, err := kapp.LoadStackConfig("large", "../../testdata/stacks.yaml")
	assert.Nil(t, err)

	expected := Values{
		"provisioner": map[interface{}]interface{}{
			"memory":    4096,
			"cpus":      4,
			"disk_size": "120g",
		},
	}

	providerImpl, err := newProviderImpl(sc.Provider)
	assert.Nil(t, err)

	actual, err := stackConfigVars(providerImpl, sc)
	assert.Nil(t, err)
	assert.Equal(t, expected, actual, "Mismatching vars")
}

func TestNewProviderError(t *testing.T) {
	actual, err := newProviderImpl("nonsense")
	assert.NotNil(t, err)
	assert.Nil(t, actual)
}

func TestNewLocalProvider(t *testing.T) {
	actual, err := newProviderImpl(LOCAL)
	assert.Nil(t, err)
	assert.Equal(t, &LocalProvider{}, actual)
}

func TestNewAWSProvider(t *testing.T) {
	actual, err := newProviderImpl(AWS)
	assert.Nil(t, err)
	assert.Equal(t, &AwsProvider{}, actual)
}
