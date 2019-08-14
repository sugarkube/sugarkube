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
	"strconv"
)

// Installs kapps with defined run units
type RunUnitInstaller struct {
	provider interfaces.IProvider
}

func (r RunUnitInstaller) Name() string {
	return RunUnit
}

// Search for a named run step in a list of them
func findStep(steps []structs.RunStep, name string) *structs.RunStep {
	for i := range steps {
		if steps[i].Name == name {
			return &steps[i]
		}
	}

	return nil
}

// Replaces run steps that call another step with the actual step they refer to
func interpolateCalls(steps []structs.RunStep, runUnits map[string]structs.RunUnit) ([]structs.RunStep, error) {

	log.Logger.Tracef("Interpolating run steps: %#v", steps)

	interpolated := make([]structs.RunStep, 0)

	for _, step := range steps {
		if step.Call != "" {
			// find the referenced step
			var targetStep *structs.RunStep
			for _, v := range runUnits {
				targetStep = findStep(v.PlanInstall, step.Call)

				if targetStep == nil {
					targetStep = findStep(v.ApplyInstall, step.Call)
				}
				if targetStep == nil {
					targetStep = findStep(v.PlanDelete, step.Call)
				}
				if targetStep == nil {
					targetStep = findStep(v.ApplyDelete, step.Call)
				}

				if targetStep != nil {
					break
				}
			}

			if targetStep == nil {
				return nil, fmt.Errorf("Unable to find run step '%s'", step.Call)
			}

			// overwrite the merge priority with the one on the call step if it's set
			var priority *uint8
			if step.MergePriority != nil {
				priority = step.MergePriority
			}

			// do a deep copy on the step so we can modify it
			step = structs.RunStep{}
			err := utils.DeepCopy(*targetStep, &step)
			if err != nil {
				return nil, errors.WithStack(err)
			}

			if priority != nil {
				step.MergePriority = priority
			}
		}

		interpolated = append(interpolated, step)
	}

	return interpolated, nil
}

// Merge steps for an action from different run units, respecting the merge priority (steps
// with a priority closer to zero will appear earlier in the returned list. Steps with no
// merge priority will appear last. Conditions on each run unit must evaluate to true to be
// included in the resulting list.
func mergeRunUnits(runUnits map[string]structs.RunUnit, action string,
	installableObj interfaces.IInstallable) ([]structs.RunStep, error) {

	log.Logger.Tracef("Merging '%s' run units for '%s': %#v", action,
		installableObj.FullyQualifiedId(), runUnits)

	steps := make([]structs.RunStep, 0)

	for k, v := range runUnits {
		allOk, err := all(runUnits[k].Conditions)
		if err != nil {
			return nil, errors.WithStack(err)
		}

		if !allOk {
			log.Logger.Infof("Some conditions for run step '%s' evaluated to false for kapp '%s'. Won't execute "+
				"run units for it.", k, installableObj.FullyQualifiedId())
			continue
		}

		log.Logger.Infof("All conditions for run step '%s' evaluated to true for kapp '%s'. "+
			"Run units will be executed for it.", k, installableObj.FullyQualifiedId())

		switch action {
		case constants.PlanInstall:
			steps = append(steps, v.PlanInstall...)
		case constants.ApplyInstall:
			steps = append(steps, v.ApplyInstall...)
		case constants.PlanDelete:
			steps = append(steps, v.PlanDelete...)
		case constants.ApplyDelete:
			steps = append(steps, v.ApplyDelete...)
		}
	}

	// interpolate calls to other steps with the actual step
	steps, err := interpolateCalls(steps, runUnits)
	if err != nil {
		return nil, errors.WithStack(err)
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

func (r RunUnitInstaller) initialise(installableObj interfaces.IInstallable,
	stackObj interfaces.IStack, action string, dryRun bool) error {

	installerVars := map[string]interface{}{
		"action":  action,
		"dry-run": dryRun,
	}

	templatedVars, err := stackObj.GetTemplatedVars(installableObj, installerVars)
	if err != nil {
		return errors.WithStack(err)
	}

	// template the kapp's descriptor
	err = installableObj.TemplateDescriptor(templatedVars)
	if err != nil {
		return errors.WithStack(err)
	}

	// all templates in the run units will now have been evaluated. So e.g. conditions should
	// just be a list of string boolean values, etc.
	runUnits := installableObj.GetDescriptor().RunUnits

	runSteps, err := mergeRunUnits(runUnits, action, installableObj)
	if err != nil {
		return errors.WithStack(err)
	}

	log.Logger.Fatalf("** Got run steps: %#v", runSteps)

	return nil
}

func (r RunUnitInstaller) PlanInstall(installableObj interfaces.IInstallable,
	stackObj interfaces.IStack, dryRun bool) error {

	err := r.initialise(installableObj, stackObj, constants.PlanInstall, dryRun)
	if err != nil {
		return errors.WithStack(err)
	}

	// todo - check for any outputs after each step and load them. this will solve
	//  passing outputs from terraform to helm

	return nil
}

func (r RunUnitInstaller) ApplyInstall(installableObj interfaces.IInstallable,
	stackObj interfaces.IStack, dryRun bool) error {

	err := r.initialise(installableObj, stackObj, constants.ApplyInstall, dryRun)
	if err != nil {
		return errors.WithStack(err)
	}

	return nil
}

func (r RunUnitInstaller) PlanDelete(installableObj interfaces.IInstallable,
	stackObj interfaces.IStack, dryRun bool) error {

	err := r.initialise(installableObj, stackObj, constants.PlanDelete, dryRun)
	if err != nil {
		return errors.WithStack(err)
	}

	return nil
}

func (r RunUnitInstaller) ApplyDelete(installableObj interfaces.IInstallable,
	stackObj interfaces.IStack, dryRun bool) error {

	err := r.initialise(installableObj, stackObj, constants.ApplyDelete, dryRun)
	if err != nil {
		return errors.WithStack(err)
	}

	return nil
}

func (r RunUnitInstaller) Clean(installableObj interfaces.IInstallable,
	stackObj interfaces.IStack, dryRun bool) error {
	return nil
}

func (r RunUnitInstaller) Output(installableObj interfaces.IInstallable,
	stackObj interfaces.IStack, dryRun bool) error {
	return nil
}

// todo - get rid of this and just return the action name
func (r RunUnitInstaller) GetVars(action string, approved bool) map[string]interface{} {
	return map[string]interface{}{
		"action":   action,
		"approved": fmt.Sprintf("%v", approved),
	}
}

// Evaluates a list of conditional strings by running them through the templater
//func evaluateConditions(conditions []string) (bool, error) {
//	results := make([]bool, 0)
//
//	for _, conditionTemplate := range conditions {
//		result := templater.RenderTemplate(conditionTemplate, )
//	}
//}

// Returns true if all conditions are true. Conditions must be parseable as booleans.
func all(conditions []string) (bool, error) {
	var boolCondition bool
	var err error
	for _, condition := range conditions {
		boolCondition, err = strconv.ParseBool(condition)
		if err != nil {
			return false, errors.WithStack(err)
		}

		if !boolCondition {
			return false, nil
		}
	}

	return true, nil
}
