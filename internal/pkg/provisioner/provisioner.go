package provisioner

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/sugarkube/sugarkube/internal/pkg/vars"
)

type Provisioner interface {
	// Method that returns all paths in a config directory relevant to the
	// target profile/cluster/region, etc. that should be searched for values
	// files to merge.
	Create(sc *vars.StackConfig, values map[string]interface{}) error
}

// Factory that creates providers
func NewProvisioner(name string) (Provisioner, error) {
	if name == "minikube" {
		return MinikubeProvisioner{}, nil
	}

	if name == "kops" {
		return KopsProvisioner{}, nil
	}

	return nil, errors.New(fmt.Sprintf("Provider '%s' doesn't exist", name))
}
