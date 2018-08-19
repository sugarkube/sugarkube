package installer

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/sugarkube/sugarkube/internal/pkg/kapp"
)

type Installer interface {
	Install(kapp *kapp.Kapp) error
}

// implemented installers
const MAKE = "make"

// Factory that creates installers
func NewInstaller(name string) (Installer, error) {
	if name == MAKE {
		return MakeInstaller{}, nil
	}

	return nil, errors.New(fmt.Sprintf("Installer '%s' doesn't exist", name))
}

func Install(i Installer, kapp *kapp.Kapp) error {
	return i.Install(kapp)
}
