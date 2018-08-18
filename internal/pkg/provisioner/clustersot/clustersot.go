package clustersot

import (
	"github.com/sugarkube/sugarkube/internal/pkg/provider"
	"github.com/sugarkube/sugarkube/internal/pkg/vars"
)

type ClusterSot interface {
	IsReady(sc *vars.StackConfig, values provider.Values) (bool, error)
}

// Uses an implementation to determine whether the cluster is ready to install kapps into
func IsReady(c ClusterSot, sc *vars.StackConfig, values provider.Values) (bool, error) {
	return c.IsReady(sc, values)
}
