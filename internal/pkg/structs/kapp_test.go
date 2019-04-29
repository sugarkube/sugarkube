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

package structs

import (
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v2"
	"testing"
)

func TestPostActions(t *testing.T) {
	input := `
post_install_actions:
  - test_1_post_inst:
      params:
        - p1
        - p2
  - atest2_pi:
pre_install_actions:
  - pre_inst_test_1:
      params:
        - p1
        - p2
  - pre_atest2:
post_delete_actions:
  - test_1_post_del:
      params:
        - p1
        - p2
pre_delete_actions:
  - pre_del_test_1:
      params:
        - p1
        - p2
  - pre_atest2:
`

	expected := KappConfig{
		PreInstallActions: []map[string]Action{
			{"pre_inst_test_1": Action{Id: "", Params: []string{"p1", "p2"}}},
			{"pre_atest2": Action{Id: "", Params: []string(nil)}},
		},
		PostInstallActions: []map[string]Action{
			{"test_1_post_inst": Action{Id: "", Params: []string{"p1", "p2"}}},
			{"atest2_pi": Action{Id: "", Params: []string(nil)}},
		},
		PreDeleteActions: []map[string]Action{
			{"pre_del_test_1": Action{Id: "", Params: []string{"p1", "p2"}}},
			{"pre_atest2": Action{Id: "", Params: []string(nil)}},
		},
		PostDeleteActions: []map[string]Action{
			{"test_1_post_del": Action{Id: "", Params: []string{"p1", "p2"}}},
		},
	}

	actual := KappConfig{}
	err := yaml.Unmarshal([]byte(input), &actual)
	assert.Nil(t, err)
	assert.Equal(t, expected, actual)
}
