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
	"github.com/sugarkube/sugarkube/internal/pkg/utils"
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
		actual, err := utils.All(input.conditions)
		assert.Nil(t, err)
		assert.Equal(t, input.expected, actual)
	}
}

func getFixtures() map[string]structs.RunUnit {
	var highest uint8 = 0
	var medium uint8 = 5
	var low uint8 = 10

	return map[string]structs.RunUnit{
		"helm": {
			PlanInstall: []structs.RunStep{
				{
					Name:          "plan-inst-3",
					Command:       "plan-inst-comm3",
					MergePriority: &low,
				},
			},
			ApplyInstall: nil,
			PlanDelete: []structs.RunStep{
				{
					Call:          "plan-install", // this should be replaced by all steps from `plan-install`
					MergePriority: &low,           // all merged steps should have this priority
				},
			},
			ApplyDelete: nil,
			Output:      nil,
			Clean:       nil,
		},
		"terraform": {
			PlanInstall: []structs.RunStep{
				{
					Name:          "st-1",
					Command:       "plan-inst-1",
					MergePriority: &highest,
				},
				{
					Name:          "st-2",
					Command:       "plan-inst-2",
					MergePriority: &medium,
				},
			},
			ApplyInstall: []structs.RunStep{
				{Call: "plan-install"},
			},
			PlanDelete: nil,
			ApplyDelete: []structs.RunStep{
				{
					Name:          "del-step-1",
					Command:       "del-command1",
					MergePriority: &medium,
				},
				{
					Call:          "plan-install/st-1", // only a single step should replace this
					MergePriority: &low,
				},
			},
			Output: nil,
			Clean:  nil,
		},
	}
}

func TestFindStep(t *testing.T) {
	var low uint8 = 10

	fixture := getFixtures()
	steps := fixture["helm"].PlanInstall
	assert.NotNil(t, steps)

	output := findStep(steps, "plan-inst-3")
	expected := structs.RunStep{
		Name:          "plan-inst-3",
		Command:       "plan-inst-comm3",
		MergePriority: &low,
	}

	assert.NotNil(t, output)

	assert.Equal(t, expected.Name, output.Name)
	assert.Equal(t, expected.Command, output.Command)
	assert.Equal(t, *expected.MergePriority, *output.MergePriority)
}

func TestFindStepInRunUnits(t *testing.T) {
	var low uint8 = 10

	fixture := getFixtures()

	output, err := findStepInRunUnits(fixture, "plan-install", "plan-inst-3")
	assert.Nil(t, err)

	expected := structs.RunStep{
		Name:          "plan-inst-3",
		Command:       "plan-inst-comm3",
		MergePriority: &low,
	}

	assert.NotNil(t, output)

	assert.Equal(t, expected.Name, output.Name)
	assert.Equal(t, expected.Command, output.Command)
	assert.Equal(t, *expected.MergePriority, *output.MergePriority)
}

func TestGetStepsInRunUnit(t *testing.T) {
	var highest uint8 = 0
	var medium uint8 = 5
	var low uint8 = 10

	fixture := getFixtures()

	output := getStepsInRunUnit(fixture, "plan-install")

	expected := []structs.RunStep{
		{
			Name:          "plan-inst-3",
			Command:       "plan-inst-comm3",
			MergePriority: &low,
		},
		{
			Name:          "st-1",
			Command:       "plan-inst-1",
			MergePriority: &highest,
		},
		{
			Name:          "st-2",
			Command:       "plan-inst-2",
			MergePriority: &medium,
		},
	}

	assert.NotNil(t, output)

	found := false
	for _, expectedStep := range expected {
		found = false

		// iterate through all the output steps because we don't care about ordering at this point
		for _, outputStep := range output {
			if expectedStep.Name == outputStep.Name {
				found = true

				assert.Equal(t, expectedStep.Name, outputStep.Name)
				assert.Equal(t, expectedStep.Command, outputStep.Command)
				assert.Equal(t, *expectedStep.MergePriority, *outputStep.MergePriority)
			}
		}

		assert.True(t, found)
	}

	assert.True(t, found)
}

func TestInterpolateCallsSingleStep(t *testing.T) {
	var medium uint8 = 5
	var low uint8 = 10

	fixture := getFixtures()
	expected := []structs.RunStep{
		{
			Name:          "del-step-1",
			Command:       "del-command1",
			MergePriority: &medium,
		},
		{
			Name:          "st-1",
			Command:       "plan-inst-1",
			MergePriority: &low,
		},
	}
	output, err := interpolateCalls(fixture["terraform"].ApplyDelete, fixture)
	assert.Nil(t, err)

	found := false
	for _, expectedStep := range expected {
		found = false

		// iterate through all the output steps because we don't care about ordering at this point
		for _, outputStep := range output {
			if expectedStep.Name == outputStep.Name {
				found = true

				assert.Equal(t, expectedStep.Name, outputStep.Name)
				assert.Equal(t, expectedStep.Command, outputStep.Command)
				assert.Equal(t, *expectedStep.MergePriority, *outputStep.MergePriority)
			}
		}

		assert.True(t, found)
	}

	assert.True(t, found)
}

func TestInterpolateCallsUnit(t *testing.T) {
	var low uint8 = 10

	fixture := getFixtures()
	expected := []structs.RunStep{
		{
			Name:          "st-1",
			Command:       "plan-inst-1",
			MergePriority: &low,
		},
		{
			Name:          "st-2",
			Command:       "plan-inst-2",
			MergePriority: &low,
		},
		{
			Name:          "plan-inst-3",
			Command:       "plan-inst-comm3",
			MergePriority: &low,
		},
	}
	output, err := interpolateCalls(fixture["helm"].PlanDelete, fixture)
	assert.Nil(t, err)

	found := false
	for _, expectedStep := range expected {
		found = false

		// iterate through all the output steps because we don't care about ordering at this point
		for _, outputStep := range output {
			if expectedStep.Name == outputStep.Name {
				found = true

				assert.Equal(t, expectedStep.Name, outputStep.Name)
				assert.Equal(t, expectedStep.Command, outputStep.Command)
				assert.Equal(t, *expectedStep.MergePriority, *outputStep.MergePriority)
			}
		}

		assert.True(t, found)
	}

	assert.True(t, found)
}
