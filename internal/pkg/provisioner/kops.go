package provisioner

import (
	"github.com/sugarkube/sugarkube/internal/pkg/log"
	"github.com/sugarkube/sugarkube/internal/pkg/provider"
	"github.com/sugarkube/sugarkube/internal/pkg/vars"
)

type KopsProvisioner struct {
	Provisioner
}

func (p KopsProvisioner) Create(sc *vars.StackConfig, values provider.Values,
	dryRun bool) error {
	log.Debugf("Creating stack with Kops and config: %#v", sc)

	// todo - implement

	return nil
}
