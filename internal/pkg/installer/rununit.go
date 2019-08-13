package installer

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/sugarkube/sugarkube/internal/pkg/constants"
	"github.com/sugarkube/sugarkube/internal/pkg/interfaces"
	"github.com/sugarkube/sugarkube/internal/pkg/log"
)

// Installs kapps with defined run units
type RunUnitInstaller struct {
	provider interfaces.IProvider
}

func (r RunUnitInstaller) Name() string {
	return RunUnit
}

func (r RunUnitInstaller) initialise(installableObj interfaces.IInstallable,
	stackObj interfaces.IStack, action string, dryRun bool) error {

	installerVars := map[string]interface{}{
		"action":  action,
		"dry-run": dryRun,
	}

	// template the kapp's descriptor
	templatedVars, err := stackObj.GetTemplatedVars(installableObj, installerVars)
	if err != nil {
		return errors.WithStack(err)
	}

	err = installableObj.TemplateDescriptor(templatedVars)
	if err != nil {
		return errors.WithStack(err)
	}

	globalConditions := installableObj.GetDescriptor().RunUnits

	log.Logger.Fatalf("** Got conditions: %#v", globalConditions)

	return nil
}

func (r RunUnitInstaller) PlanInstall(installableObj interfaces.IInstallable,
	stackObj interfaces.IStack, dryRun bool) error {

	err := r.initialise(installableObj, stackObj, constants.PlanInstall, dryRun)
	if err != nil {
		return errors.WithStack(err)
	}

	return nil
}

func (r RunUnitInstaller) ApplyInstall(installableObj interfaces.IInstallable,
	stackObj interfaces.IStack, dryRun bool) error {
	return nil
}

func (r RunUnitInstaller) PlanDelete(installableObj interfaces.IInstallable,
	stackObj interfaces.IStack, dryRun bool) error {
	return nil
}

func (r RunUnitInstaller) ApplyDelete(installableObj interfaces.IInstallable,
	stackObj interfaces.IStack, dryRun bool) error {
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

// Return 'true' if all conditions are true
func all(conditions []bool) bool {
	for _, condition := range conditions {
		if !condition {
			return false
		}
	}

	return true
}
