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

package installer

import (
	"github.com/stretchr/testify/assert"
	"github.com/sugarkube/sugarkube/internal/pkg/constants"
	"github.com/sugarkube/sugarkube/internal/pkg/log"
	"github.com/sugarkube/sugarkube/internal/pkg/structs"
	"testing"
)

func init() {
	log.ConfigureLogger("debug", false)
}

func TestMergeRunUnits(t *testing.T) {
	var highest uint8 = 0
	var medium uint8 = 5
	var low uint8 = 10

	inputs := []struct {
		runUnits map[string]structs.RunUnit
		expected []string
	}{
		{
			map[string]structs.RunUnit{
				"helm": {
					Conditions: []string{"true"},
					PlanInstall: []structs.RunStep{
						{
							Name:          "should-be-penultimate",
							MergePriority: &low,
						},
						{
							Name:          "should-be-first",
							MergePriority: &highest,
						},
						{
							Name: "should-be-last", // no priority == lowest priority
						},
						{
							Name:          "should-be-second",
							MergePriority: &medium,
						},
					},
				},
			},

			[]string{
				"should-be-first",
				"should-be-second",
				"should-be-penultimate",
				"should-be-last",
			},
		},
		{
			map[string]structs.RunUnit{
				"helm": {
					Conditions: []string{"true"},
					PlanInstall: []structs.RunStep{
						{
							Name:          "should-be-penultimate",
							MergePriority: &low,
						},
						{
							Name:          "should-be-second",
							MergePriority: &medium,
						},
					},
				},
				"terraform": {
					Conditions: []string{"true"},
					PlanInstall: []structs.RunStep{
						{
							Name:          "should-be-first",
							MergePriority: &highest,
						},
						{
							Name: "should-be-last", // no priority == lowest priority
						},
					},
				},
				"missing": {
					Conditions: []string{"false"}, // false, so should be filtered out
					PlanInstall: []structs.RunStep{
						{
							Name:          "should-be-missing",
							MergePriority: &highest,
						},
					},
				},
			},

			[]string{
				"should-be-first",
				"should-be-second",
				"should-be-penultimate",
				"should-be-last",
			},
		},
	}

	installableObj := MockInstallable{Name: "mock-installable"}

	for _, input := range inputs {
		actual, err := mergeRunUnits(input.runUnits, constants.PlanInstall, installableObj)
		assert.Nil(t, err)

		actualNames := make([]string, 0)
		for _, step := range actual {
			actualNames = append(actualNames, step.Name)
		}

		assert.Equal(t, input.expected, actualNames)
	}
}

func TestAll(t *testing.T) {
	inputs := []struct {
		conditions []string
		expected   bool
	}{
		{[]string{"true", "true"}, true},
		{[]string{"true", "true", "1"}, true},
		{[]string{"true", "false"}, false},
		{[]string{"true", "false", "true"}, false},
	}

	for _, input := range inputs {
		actual, err := all(input.conditions)
		assert.Nil(t, err)
		assert.Equal(t, input.expected, actual)
	}
}
