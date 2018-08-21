package cacher

import (
	"github.com/pkg/errors"
	"github.com/sugarkube/sugarkube/internal/pkg/kapp"
	"github.com/sugarkube/sugarkube/internal/pkg/log"
	"os"
	"path/filepath"
)

const CACHE_DIR = "_cache"

// Build a cache for a manifest into a directory
func CacheManifest(manifest kapp.Manifest, cacheDir string) error {

	// create a directory to cache all kapps in this manifest in
	manifestCacheDir := filepath.Join(cacheDir, manifest.Id)

	log.Debugf("Creating manifest cache dir: %s", manifestCacheDir)
	err := os.MkdirAll(manifestCacheDir, 0755)
	if err != nil {
		return errors.WithStack(err)
	}

	// acquire each kapp and cache it
	for _, kapp := range manifest.Kapps {
		// create a cache directory for the kapp
		kappCacheDir := filepath.Join(manifestCacheDir, kapp.Id, CACHE_DIR)

		log.Debugf("Creating kapp cache dir: %s", kappCacheDir)
		err := os.MkdirAll(manifestCacheDir, 0755)
		if err != nil {
			return errors.WithStack(err)
		}

		// acquire each source
		for _, acquirer := range kapp.Sources {
			sourceDest := filepath.Join(kappCacheDir, acquirer.Id())
			acquirer.Acquire(sourceDest)
		}
	}
}
