package installer

import "github.com/sugarkube/sugarkube/internal/pkg/interfaces"

// Installs kapps with defined run units
type RunUnitInstaller struct {
	provider interfaces.IProvider
}

func (r RunUnitInstaller) PlanInstall(installableObj interfaces.IInstallable,
	stack interfaces.IStack, dryRun bool) error {
	return nil
}

func (r RunUnitInstaller) ApplyInstall(installableObj interfaces.IInstallable,
	stack interfaces.IStack, dryRun bool) error {
	return nil
}

func (r RunUnitInstaller) PlanDelete(installableObj interfaces.IInstallable,
	stack interfaces.IStack, dryRun bool) error {
	return nil
}

func (r RunUnitInstaller) ApplyDelete(installableObj interfaces.IInstallable,
	stack interfaces.IStack, dryRun bool) error {
	return nil
}

func (r RunUnitInstaller) Clean(installableObj interfaces.IInstallable,
	stack interfaces.IStack, dryRun bool) error {
	return nil
}

func (r RunUnitInstaller) Output(installableObj interfaces.IInstallable,
	stack interfaces.IStack, dryRun bool) error {
	return nil
}

func (r RunUnitInstaller) Name() string {
	return RunUnit
}

func (r RunUnitInstaller) GetVars(action string, approved bool) map[string]interface{} {
	return nil
}
