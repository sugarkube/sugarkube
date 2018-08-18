package provisioner

import (
	"github.com/sugarkube/sugarkube/internal/pkg/log"
	"github.com/sugarkube/sugarkube/internal/pkg/vars"
)

type MinikubeProvisioner struct {
	Provisioner
}

func (p MinikubeProvisioner) Create(sc *vars.StackConfig, values map[string]interface{}) error {

	log.Debugf("Creating stack with Minikube and config: %#v", sc)

	return nil
}
