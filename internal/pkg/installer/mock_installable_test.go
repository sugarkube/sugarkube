package installer

import (
	"github.com/sugarkube/sugarkube/internal/pkg/acquirer"
	"github.com/sugarkube/sugarkube/internal/pkg/interfaces"
	"github.com/sugarkube/sugarkube/internal/pkg/structs"
)

type MockInstallable struct {
	Name string
}

func (m MockInstallable) Id() string {
	return m.Name
}
func (m MockInstallable) FullyQualifiedId() string {
	return ""
}
func (m MockInstallable) ManifestId() string {
	return ""
}
func (m MockInstallable) State() string {
	return ""
}
func (m MockInstallable) HasActions() bool {
	return false
}
func (m MockInstallable) PreInstallActions() []structs.Action {
	return nil
}
func (m MockInstallable) PreDeleteActions() []structs.Action {
	return nil
}
func (m MockInstallable) PostInstallActions() []structs.Action {
	return nil
}
func (m MockInstallable) PostDeleteActions() []structs.Action {
	return nil
}
func (m MockInstallable) GetDescriptor() structs.KappDescriptorWithMaps {
	return structs.KappDescriptorWithMaps{}
}
func (m MockInstallable) LoadConfigFile(workspaceDir string) error {
	return nil
}
func (m MockInstallable) SetWorkspaceDir(workspaceDir string) error {
	return nil
}
func (m MockInstallable) GetCacheDir() string {
	return ""
}
func (m MockInstallable) GetConfigFileDir() string {
	return ""
}
func (m MockInstallable) Acquirers() (map[string]acquirer.Acquirer, error) {
	return nil, nil
}
func (m MockInstallable) TemplateDescriptor(templateVars map[string]interface{}) error {
	return nil
}
func (m MockInstallable) GetCliArgs(installerName string, command string) []string {
	return nil
}
func (m MockInstallable) GetEnvVars() map[string]interface{} {
	return nil
}
func (m MockInstallable) Vars(stack interfaces.IStack) (map[string]interface{}, error) {
	return nil, nil
}
func (m MockInstallable) AddDescriptor(config structs.KappDescriptorWithMaps, prepend bool) error {
	return nil
}
func (m MockInstallable) RenderTemplates(templateVars map[string]interface{}, stackConfig interfaces.IStackConfig, dryRun bool) error {
	return nil
}
func (m MockInstallable) GetOutputs(ignoreMissing bool, dryRun bool) (map[string]interface{}, error) {
	return nil, nil
}
func (m MockInstallable) HasOutputs() bool {
	return false
}
func (m MockInstallable) GetLocalRegistry() interfaces.IRegistry {
	return nil
}
func (m MockInstallable) SetLocalRegistry(registry interfaces.IRegistry) {
}
