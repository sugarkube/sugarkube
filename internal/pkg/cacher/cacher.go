package cacher

import (
	"github.com/pkg/errors"
	"github.com/sugarkube/sugarkube/internal/pkg/acquirer"
	"github.com/sugarkube/sugarkube/internal/pkg/kapp"
	"github.com/sugarkube/sugarkube/internal/pkg/log"
	"os"
	"path/filepath"
	"strings"
)

const CACHE_DIR = ".sugarkube"

// Build a cache for a manifest into a directory
func CacheManifest(manifest kapp.Manifest, cacheDir string, dryRun bool) error {

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
		kappDir := filepath.Join(manifestCacheDir, kapp.Id)
		kappCacheDir := filepath.Join(kappDir, CACHE_DIR)

		log.Debugf("Creating kapp cache dir: %s", kappCacheDir)
		err := os.MkdirAll(kappCacheDir, 0755)
		if err != nil {
			return errors.WithStack(err)
		}

		err = acquireSource(kapp.Sources, kappDir, kappCacheDir, dryRun)
		if err != nil {
			return errors.WithStack(err)
		}
	}

	return nil
}

// Acquires each source and symlinks it to the target path in the cache directory.
// Runs all acquirers in parallel.
func acquireSource(acquirers []acquirer.Acquirer, kappDir string,
	kappCacheDir string, dryRun bool) error {
	doneCh := make(chan bool)
	errCh := make(chan error)

	for _, acquirerImpl := range acquirers {
		go func(a acquirer.Acquirer) {
			acquirerId, err := a.Id()
			if err != nil {
				errCh <- errors.Wrap(err, "Invalid acquirer ID")
			}

			sourceDest := filepath.Join(kappCacheDir, acquirerId)

			if dryRun {
				log.Debugf("Dry run: Would acquire source into: %s", sourceDest)
			} else {
				err := a.Acquire(sourceDest)
				if err != nil {
					errCh <- errors.WithStack(err)
				}
			}

			sourcePath := filepath.Join(sourceDest, a.Path())
			sourcePath = strings.TrimPrefix(sourcePath, kappDir)
			sourcePath = strings.TrimPrefix(sourcePath, "/")

			symLinkTarget := filepath.Join(kappDir, a.Name())

			if dryRun {
				log.Debugf("Dry run. Would symlink cached source %s to %s", sourcePath, symLinkTarget)
			} else {
				if _, err := os.Stat(filepath.Join(kappDir, sourcePath)); err != nil {
					errCh <- errors.Wrapf(err, "Symlink source '%s' doesn't exist", sourcePath)
				}

				log.Debugf("Symlinking cached source %s to %s", sourcePath, symLinkTarget)
				err := os.Symlink(sourcePath, symLinkTarget)
				if err != nil {
					errCh <- errors.Wrapf(err, "Error symlinking kapp source")
				}
			}

			doneCh <- true
		}(acquirerImpl)
	}

	for success := 0; success < len(acquirers); success++ {
		select {
		case err := <-errCh:
			close(doneCh)
			log.Warnf("Error in acquirer goroutines: %s", err)
			return errors.Wrap(err, "Error running acquirer in goroutine")
		case <-doneCh:
			log.Debugf("%d acquirers successfully completed", success+1)
		}
	}

	return nil
}
