package installer

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/sugarkube/sugarkube/internal/pkg/constants"
	"github.com/sugarkube/sugarkube/internal/pkg/interfaces"
	"github.com/sugarkube/sugarkube/internal/pkg/log"
	"github.com/sugarkube/sugarkube/internal/pkg/structs"
	"github.com/sugarkube/sugarkube/internal/pkg/utils"
	"math"
	"sort"
	"strings"
)

// Installs kapps with defined run units
type RunUnitInstaller struct {
	provider interfaces.IProvider
}

const maxInterpolationRecursions = 5

func (r RunUnitInstaller) Name() string {
	return RunUnit
}

// Search for a named run step (with no run unit prefix) in a list of them
func findStep(steps []structs.RunStep, name string) *structs.RunStep {
	for i := range steps {
		if steps[i].Name == name {
			return &steps[i]
		}
	}

	return nil
}

// Searches for a run step in a specific run unit
func findStepInRunUnits(runUnits map[string]structs.RunUnit, unitName string, stepName string) (*structs.RunStep, error) {
	var targetStep *structs.RunStep
	for _, v := range runUnits {
		switch unitName {
		case constants.PlanInstall:
			targetStep = findStep(v.PlanInstall, stepName)
		case constants.ApplyInstall:
			targetStep = findStep(v.ApplyInstall, stepName)
		case constants.PlanDelete:
			targetStep = findStep(v.PlanDelete, stepName)
		case constants.ApplyDelete:
			targetStep = findStep(v.ApplyDelete, stepName)
		case constants.Output:
			targetStep = findStep(v.Output, stepName)
		case constants.Clean:
			targetStep = findStep(v.Clean, stepName)
		}

		if targetStep != nil {
			intermediate := setWorkingDir(v, []structs.RunStep{*targetStep})
			if len(intermediate) != 1 {
				return nil, fmt.Errorf("Error setting working directory on run step: %#v", *targetStep)
			}

			targetStep = &intermediate[0]
			break
		}
	}

	if targetStep == nil {
		return nil, fmt.Errorf("Unable to find run step '%s/%s'", unitName, stepName)
	}

	return targetStep, nil
}

// Returns all steps for the named run unit in a map of run units
func getStepsInRunUnit(runUnits map[string]structs.RunUnit, unitName string) []structs.RunStep {
	result := make([]structs.RunStep, 0)
	for _, v := range runUnits {
		var steps []structs.RunStep
		switch unitName {
		case constants.PlanInstall:
			steps = v.PlanInstall
		case constants.ApplyInstall:
			steps = v.ApplyInstall
		case constants.PlanDelete:
			steps = v.PlanDelete
		case constants.ApplyDelete:
			steps = v.ApplyDelete
		case constants.Output:
			steps = v.Output
		case constants.Clean:
			steps = v.Clean
		}

		steps = setWorkingDir(v, steps)

		result = append(result, steps...)
	}

	return result
}

// Replaces run steps that call another step with the actual step they refer to, recursing up to a maximum number of times
func interpolateCalls(steps []structs.RunStep, runUnits map[string]structs.RunUnit, maxRecursions uint8) ([]structs.RunStep, error) {

	log.Logger.Tracef("Interpolating run steps: %#v", steps)

	interpolated := make([]structs.RunStep, 0)

	for _, step := range steps {
		if step.Call != "" {
			if step.Command != "" {
				return nil, fmt.Errorf("A run step cannot have 'call' and 'command' blocks")
			}

			targetSteps := make([]structs.RunStep, 0)

			// the step might either be the name of an entire run unit, or formatted 'unit/step' to refer to a single step
			targetParts := strings.Split(step.Call, constants.CallSeparator)

			if len(targetParts) == 1 {
				// add all the steps from the target (which themselves may contain calls that need interpolating)
				targetSteps = getStepsInRunUnit(runUnits, targetParts[0])
			} else if len(targetParts) == 2 {
				// add a single specific step
				targetStep, err := findStepInRunUnits(runUnits, targetParts[0], targetParts[1])

				if err != nil {
					return nil, errors.WithStack(err)
				}

				targetSteps = append(targetSteps, *targetStep)
			}

			// massage all the steps we've found and add them to the array in place of the original call step
			for _, targetStep := range targetSteps {
				// overwrite the merge priority with the one on the call step if it's set
				var priority *uint8
				if step.MergePriority != nil {
					priority = step.MergePriority
				}

				// do a deep copy on the step so we can modify it
				interpolatedStep := structs.RunStep{}
				err := utils.DeepCopy(targetStep, &interpolatedStep)
				if err != nil {
					return nil, errors.WithStack(err)
				}

				if priority != nil {
					interpolatedStep.MergePriority = priority
				}

				interpolated = append(interpolated, interpolatedStep)
			}
		} else {
			interpolated = append(interpolated, step)
		}
	}

	// recurse if any interpolated steps contain call blocks
	if callsToInterpolate(interpolated) {
		if maxRecursions <= 0 {
			return nil, errors.New("Recursion limit reached interpolating run step calls")
		}

		var err error
		// interpolate calls to other steps with the actual step
		interpolated, err = interpolateCalls(interpolated, runUnits, maxRecursions-1)
		if err != nil {
			return nil, errors.WithStack(err)
		}
	}

	return interpolated, nil
}

// Returns a boolean indicating whether any of the input run steps contain calls to other steps that need interpolating
func callsToInterpolate(runSteps []structs.RunStep) bool {
	for _, step := range runSteps {
		if step.Call != "" {
			return true
		}
	}

	return false
}

// Default to the run unit's working dir if the step doesn't define its own
func setWorkingDir(runUnit structs.RunUnit, runSteps []structs.RunStep) []structs.RunStep {

	// use the unit's working dir if none was defined on the step itself
	for i := range runSteps {
		if runSteps[i].WorkingDir == "" {
			runSteps[i].WorkingDir = runUnit.WorkingDir
		}
	}

	return runSteps
}

// Merge steps for an action from different run units, respecting the merge priority (steps
// with a priority closer to zero will appear earlier in the returned list. Steps with no
// merge priority will appear last. Conditions on each run unit must evaluate to true to be
// included in the resulting list.
func mergeRunUnits(runUnits map[string]structs.RunUnit, action string,
	installableObj interfaces.IInstallable) ([]structs.RunStep, error) {

	log.Logger.Tracef("Merging '%s' run units for '%s': %#v", action,
		installableObj.FullyQualifiedId(), runUnits)

	var err error
	steps := make([]structs.RunStep, 0)

	for k, v := range runUnits {
		allOk, err := utils.All(runUnits[k].Conditions)
		if err != nil {
			return nil, errors.WithStack(err)
		}

		if !allOk {
			log.Logger.Infof("Some conditions for run step '%s' evaluated to false for kapp '%s'. Won't execute "+
				"run units for it.", k, installableObj.FullyQualifiedId())
			continue
		}

		log.Logger.Infof("All conditions for run unit '%s' evaluated to true for kapp '%s'. "+
			"Run steps will be executed for it.", k, installableObj.FullyQualifiedId())

		switch action {
		case constants.PlanInstall:
			runSteps := setWorkingDir(v, v.PlanInstall)
			steps = append(steps, runSteps...)
		case constants.ApplyInstall:
			runSteps := setWorkingDir(v, v.ApplyInstall)
			steps = append(steps, runSteps...)
		case constants.PlanDelete:
			runSteps := setWorkingDir(v, v.PlanDelete)
			steps = append(steps, runSteps...)
		case constants.ApplyDelete:
			runSteps := setWorkingDir(v, v.ApplyDelete)
			steps = append(steps, runSteps...)
		}
	}

	if callsToInterpolate(steps) {
		// interpolate calls to other steps with the actual step
		steps, err = interpolateCalls(steps, runUnits, maxInterpolationRecursions)
		if err != nil {
			return nil, errors.WithStack(err)
		}
	}

	// use a temporary variable because we can't use the address of a constant directly
	var maxPriority uint8 = math.MaxUint8

	// now we have a list of all run steps to execute, add a maximum merge priority to those
	// where it's not defined to simplify sorting
	for i := 0; i < len(steps); i++ {
		if steps[i].MergePriority == nil {
			steps[i].MergePriority = &maxPriority
		}
	}

	// now sort based on merge priority
	sort.Slice(steps, func(i, j int) bool {
		log.Logger.Tracef("Sorting run steps for %s: %s priority=%d vs %s priority=%d",
			installableObj.FullyQualifiedId(), steps[i].Name, *steps[i].MergePriority,
			steps[j].Name, *steps[j].MergePriority)
		return *steps[i].MergePriority < *steps[j].MergePriority
	})

	log.Logger.Debugf("Sorted run steps for '%s' on '%s' are: %#v", action,
		installableObj.FullyQualifiedId(), steps)

	return steps, nil
}

func (r RunUnitInstaller) getRunSteps(installableObj interfaces.IInstallable,
	stackObj interfaces.IStack, action string, dryRun bool) ([]structs.RunStep, error) {

	installerVars := map[string]interface{}{
		"action":  action,
		"dry-run": dryRun,
	}

	templatedVars, err := stackObj.GetTemplatedVars(installableObj, installerVars)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	// template the kapp's descriptor
	err = installableObj.TemplateDescriptor(templatedVars)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	// all templates in the run units will now have been evaluated. So e.g. conditions should
	// just be a list of string boolean values, etc.
	runUnits := installableObj.GetDescriptor().RunUnits

	// todo - validate that there aren't multiple run steps with the same name for a given run unit (which would mess up calling
	//  run steps)

	runSteps, err := mergeRunUnits(runUnits, action, installableObj)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	log.Logger.Debugf("Calculated '%s' run steps for '%s': %#v", action, installableObj.FullyQualifiedId(),
		runSteps)

	return runSteps, nil
}

func (r RunUnitInstaller) PlanInstall(installableObj interfaces.IInstallable,
	stackObj interfaces.IStack, dryRun bool) ([]structs.RunStep, error) {

	return r.getRunSteps(installableObj, stackObj, constants.PlanInstall, dryRun)
}

func (r RunUnitInstaller) ApplyInstall(installableObj interfaces.IInstallable,
	stackObj interfaces.IStack, dryRun bool) ([]structs.RunStep, error) {

	return r.getRunSteps(installableObj, stackObj, constants.ApplyInstall, dryRun)
}

func (r RunUnitInstaller) PlanDelete(installableObj interfaces.IInstallable,
	stackObj interfaces.IStack, dryRun bool) ([]structs.RunStep, error) {

	return r.getRunSteps(installableObj, stackObj, constants.PlanDelete, dryRun)
}

func (r RunUnitInstaller) ApplyDelete(installableObj interfaces.IInstallable,
	stackObj interfaces.IStack, dryRun bool) ([]structs.RunStep, error) {

	return r.getRunSteps(installableObj, stackObj, constants.ApplyDelete, dryRun)
}

func (r RunUnitInstaller) Clean(installableObj interfaces.IInstallable,
	stackObj interfaces.IStack, dryRun bool) ([]structs.RunStep, error) {

	return r.getRunSteps(installableObj, stackObj, constants.Clean, dryRun)
}

func (r RunUnitInstaller) Output(installableObj interfaces.IInstallable,
	stackObj interfaces.IStack, dryRun bool) ([]structs.RunStep, error) {

	return r.getRunSteps(installableObj, stackObj, constants.Output, dryRun)
}

// todo - get rid of this and just return the action name
func (r RunUnitInstaller) GetVars(action string, dryRun bool) map[string]interface{} {
	return map[string]interface{}{
		"action":  action,
		"dry-run": dryRun,
	}
}
