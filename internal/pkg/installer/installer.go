package installer

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/sugarkube/sugarkube/internal/pkg/kapp"
	"github.com/sugarkube/sugarkube/internal/pkg/log"
	"github.com/sugarkube/sugarkube/internal/pkg/provider"
)

type Installer interface {
	install(kappObj *kapp.Kapp, stackConfig *kapp.StackConfig) error
	destroy(kappObj *kapp.Kapp, stackConfig *kapp.StackConfig) error
}

// implemented installers
const MAKE = "make"

// Factory that creates installers
func NewInstaller(name string, providerImpl provider.Provider) (Installer, error) {
	if name == MAKE {
		return MakeInstaller{
			provider: providerImpl,
		}, nil
	}

	return nil, errors.New(fmt.Sprintf("Installer '%s' doesn't exist", name))
}

// Installs a kapp by delegating to an Installer implementation
func Install(i Installer, kappObj *kapp.Kapp, stackConfig *kapp.StackConfig) error {
	log.Infof("Installing kapp '%s'...", kappObj.Id)
	return i.install(kappObj, stackConfig)
}

// Destroys a kapp by delegating to an Installer implementation
func Destroy(i Installer, kappObj *kapp.Kapp, stackConfig *kapp.StackConfig) error {
	log.Infof("Destroying kapp '%s'...", kappObj.Id)
	return i.destroy(kappObj, stackConfig)
}
