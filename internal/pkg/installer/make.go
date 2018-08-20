package installer

import (
	"github.com/sugarkube/sugarkube/internal/pkg/kapp"
)

// Installs kapps with make
type MakeInstaller struct{}

func (i MakeInstaller) Install(kapp *kapp.Kapp) error {
	panic("to do: Implement MakeInstaller")
	return nil
}
