package clustersot

import (
	"github.com/pkg/errors"
	"github.com/sugarkube/sugarkube/internal/pkg/provider"
	"github.com/sugarkube/sugarkube/internal/pkg/vars"
)

type ClusterSot interface {
	IsOnline(sc *vars.StackConfig, values provider.Values) (bool, error)
	IsReady(sc *vars.StackConfig, values provider.Values) (bool, error)
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
		return true, nil
	}

	return false, errors.New("Timed out waiting for the cluster to come online")
}

// Uses an implementation to determine whether the cluster is ready to install kapps into
func IsReady(c ClusterSot, sc *vars.StackConfig, values provider.Values) (bool, error) {
	return c.IsReady(sc, values)
}
