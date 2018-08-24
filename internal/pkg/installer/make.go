package installer

import (
	"github.com/sugarkube/sugarkube/internal/pkg/kapp"
)

// Installs kapps with make
type MakeInstaller struct{}

func (i MakeInstaller) install(kapp *kapp.Kapp, stackConfig *kapp.StackConfig) error {
	return nil
}

func (i MakeInstaller) destroy(kapp *kapp.Kapp, stackConfig *kapp.StackConfig) error {
	return nil
}
