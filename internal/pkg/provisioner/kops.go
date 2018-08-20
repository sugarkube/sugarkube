package provisioner

import (
	"github.com/pkg/errors"
	"github.com/sugarkube/sugarkube/internal/pkg/log"
	"github.com/sugarkube/sugarkube/internal/pkg/provider"
	"github.com/sugarkube/sugarkube/internal/pkg/provisioner/clustersot"
	"github.com/sugarkube/sugarkube/internal/pkg/vars"
)

type KopsProvisioner struct {
	clusterSot clustersot.ClusterSot
}

func (p KopsProvisioner) Create(sc *vars.StackConfig, values provider.Values,
	dryRun bool) error {
	log.Debugf("Creating stack with Kops and config: %#v", sc)

	panic("not implemented")
}

func (p KopsProvisioner) ClusterSot() (clustersot.ClusterSot, error) {
	if p.clusterSot == nil {
		clusterSot, err := clustersot.NewClusterSot(clustersot.KUBECTL)
		if err != nil {
			return nil, errors.WithStack(err)
		}

		p.clusterSot = clusterSot
	}

	return p.clusterSot, nil
}

func (p KopsProvisioner) IsAlreadyOnline(sc *vars.StackConfig, values provider.Values) (bool, error) {
	panic("not implemented")
}

// No-op function, required to fully implement the Provisioner interface
func (p KopsProvisioner) Update(sc *vars.StackConfig, values provider.Values) error {
	panic("not implemented")
}
