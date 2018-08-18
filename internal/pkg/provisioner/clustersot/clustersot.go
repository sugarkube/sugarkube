package clustersot

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/sugarkube/sugarkube/internal/pkg/provider"
	"github.com/sugarkube/sugarkube/internal/pkg/vars"
)

type ClusterSot interface {
	IsOnline(sc *vars.StackConfig, values provider.Values) (bool, error)
	IsReady(sc *vars.StackConfig, values provider.Values) (bool, error)
}

// Implemented ClusterSot names
const KUBECTL = "kubectl"

// Factory that creates ClusterSots
func NewClusterSot(name string) (ClusterSot, error) {
	if name == KUBECTL {
		return KubeCtlClusterSot{}, nil
	}

	return nil, errors.New(fmt.Sprintf("ClusterSot '%s' doesn't exist", name))
}

// Uses an implementation to determine whether the cluster is reachable/online, but it
// may not be ready to install Kapps into yet.
func IsOnline(c ClusterSot, sc *vars.StackConfig, values provider.Values) (bool, error) {
	online, err := c.IsOnline(sc, values)
	if err != nil {
		return false, errors.WithStack(err)
	}

	if online {
		sc.Status.IsOnline = true
	}

	return online, nil
}

// Uses an implementation to determine whether the cluster is ready to install kapps into
func IsReady(c ClusterSot, sc *vars.StackConfig, values provider.Values) (bool, error) {
	return c.IsReady(sc, values)
}
