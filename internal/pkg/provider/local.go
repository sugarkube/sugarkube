package provider

import (
	"github.com/sugarkube/sugarkube/internal/pkg/vars"
	"path/filepath"
)

type LocalProvider struct {
	Provider
}

const providerName = "local"
const profileDir = "profiles"
const clusterDir = "clusters"

// Returns directories to look for values files in specific to this provider
func (p LocalProvider) VarsDirs(sc *vars.StackConfig) []string {

	paths := make([]string, 0)

	for _, path := range sc.VarsFilesDirs {
		paths = append(paths, filepath.Join(path))
		paths = append(paths, filepath.Join(path, providerName))
		paths = append(paths, filepath.Join(path, providerName, profileDir))
		paths = append(paths, filepath.Join(path, providerName, profileDir, sc.Profile))
		paths = append(paths, filepath.Join(path, providerName, profileDir, sc.Profile, clusterDir))
		paths = append(paths, filepath.Join(path, providerName, profileDir, sc.Profile, clusterDir, sc.Cluster))
	}

	return paths
}
