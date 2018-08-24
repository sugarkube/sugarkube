package provider

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/sugarkube/sugarkube/internal/pkg/kapp"
	"github.com/sugarkube/sugarkube/internal/pkg/log"
	"os"
	"path/filepath"
)

type LocalProvider struct {
	stackConfigVars Values
}

const PROVIDER_NAME = "local"
const PROFILE_DIR = "profiles"
const CLUSTER_DIR = "clusters"

// Associate provider variables with the provider
func (p LocalProvider) setVars(values Values) {
	p.stackConfigVars = values
}

// Returns the variables loaded by the Provider
func (p LocalProvider) GetVars() Values {
	return p.stackConfigVars
}

// Returns directories to look for values files in specific to this provider
func (p LocalProvider) varsDirs(sc *kapp.StackConfig) ([]string, error) {

	paths := make([]string, 0)

	prefix := sc.Dir()

	for _, path := range sc.VarsFilesDirs {
		// prepend the directory of the stack config file if the path is relative
		if !filepath.IsAbs(path) {
			path = filepath.Join(prefix, path)
			log.Debugf("Prepended dir of stack config to relative path. New path %s", path)
		}

		profileDir := filepath.Join(path, PROVIDER_NAME, PROFILE_DIR, sc.Profile)
		clusterDir := filepath.Join(path, PROVIDER_NAME, PROFILE_DIR, sc.Profile, CLUSTER_DIR, sc.Cluster)

		if err := abortIfNotDir(profileDir,
			fmt.Sprintf("No profile directory found at %s", profileDir)); err != nil {
			return nil, err
		}

		if err := abortIfNotDir(clusterDir,
			fmt.Sprintf("No cluster directory found at %s", clusterDir)); err != nil {
			return nil, err
		}

		paths = append(paths, filepath.Join(path))
		paths = append(paths, filepath.Join(path, PROVIDER_NAME))
		paths = append(paths, filepath.Join(path, PROVIDER_NAME, PROFILE_DIR))
		paths = append(paths, profileDir)
		paths = append(paths, filepath.Join(path, PROVIDER_NAME, PROFILE_DIR, sc.Profile, CLUSTER_DIR))
		paths = append(paths, clusterDir)
	}

	return paths, nil
}

// Returns an error if the given path doesn't exist or isn't a directory
func abortIfNotDir(path string, errorMessage string) error {
	info, err := os.Stat(path)
	if err != nil {
		return errors.Wrap(err, errorMessage)
	}

	if !info.IsDir() {
		return errors.New(fmt.Sprintf("Path '%s' is not a directory", path))
	}

	return nil
}
