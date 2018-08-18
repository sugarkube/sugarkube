package provider

import (
	"github.com/sugarkube/sugarkube/internal/pkg/vars"
	"path/filepath"
)

type LocalProvider struct {
	Provider
}

const profileDir = "profiles"
const clusterDir = "clusters"

func (p LocalProvider) VarsDirs(sc *vars.StackConfig) []string {

	paths := make([]string, 0)

	for _, path := range sc.VarsFilesDirs {
		paths = append(paths, filepath.Join(path))
		paths = append(paths, filepath.Join(path, profileDir))
		paths = append(paths, filepath.Join(path, profileDir, sc.Profile))
		paths = append(paths, filepath.Join(path, profileDir, sc.Profile, clusterDir))
		paths = append(paths, filepath.Join(path, profileDir, sc.Profile, clusterDir, sc.Cluster))
	}

	return paths
}
