package installer

import (
	"github.com/sugarkube/sugarkube/internal/pkg/kapp"
)

// Installs kapps with make
type MakeInstaller struct{}

func (i MakeInstaller) Install(kapp *kapp.Kapp, stackConfig *kapp.StackConfig) error {
	return nil
}

func (i MakeInstaller) Destroy(kapp *kapp.Kapp, stackConfig *kapp.StackConfig) error {
	return nil
}
