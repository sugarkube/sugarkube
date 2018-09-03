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

func TestLocalVarsDirs(t *testing.T) {
	sc, err := kapp.LoadStackConfig("large", "../../testdata/stacks.yaml")
	assert.Nil(t, err)

	expected := []string{
		"../../testdata/stacks",
		"../../testdata/stacks/local",
		"../../testdata/stacks/local/profiles",
		"../../testdata/stacks/local/profiles/local",
		"../../testdata/stacks/local/profiles/local/clusters",
		"../../testdata/stacks/local/profiles/local/clusters/large",
	}

	provider := LocalProvider{}
	actual, err := provider.varsDirs(sc)
	assert.Nil(t, err)

	assert.Equal(t, expected, actual, "Incorrect vars dirs returned")
}
