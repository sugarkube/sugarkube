package provisioner

import (
	"github.com/sugarkube/sugarkube/internal/pkg/log"
	"github.com/sugarkube/sugarkube/internal/pkg/provider"
	"github.com/sugarkube/sugarkube/internal/pkg/vars"
)

type MinikubeProvisioner struct {
	Provisioner
}

func (p MinikubeProvisioner) Create(sc *vars.StackConfig, values *provider.Values) error {

	log.Debugf("Creating stack with Minikube and config: %#v", sc)

	return nil
}

func (p MinikubeProvisioner) IsOnline(sc *vars.StackConfig, values *provider.Values) (bool, error) {
	panic("not implemented")
}

func (p MinikubeProvisioner) Update(sc *vars.StackConfig, values *provider.Values) error {
	panic("not implemented")
}
