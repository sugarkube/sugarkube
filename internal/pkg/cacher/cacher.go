/*
 * Copyright 2018 The Sugarkube Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

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

// Returns the cache dir for a manifest
func GetManifestCachePath(cacheDir string, manifest kapp.Manifest) string {
	return filepath.Join(cacheDir, manifest.Id())
}

// Returns the path of a kapp's cache dir where the different sources are
// checked out to
func getKappCachePath(kappRootPath string) string {
	return filepath.Join(kappRootPath, CACHE_DIR)
}

// Build a cache for a manifest into a directory
func CacheManifest(manifest kapp.Manifest, cacheDir string, dryRun bool) error {

	// create a directory to cache all kapps in this manifest in
	manifestCacheDir := GetManifestCachePath(cacheDir, manifest)

	if _, err := os.Stat(manifestCacheDir); err != nil {
		if os.IsNotExist(err) {
			log.Logger.Infof("Creating manifest cache dir '%s'", manifestCacheDir)
			err := os.MkdirAll(manifestCacheDir, 0755)
			if err != nil {
				return errors.WithStack(err)
			}
		} else {
			return errors.Wrapf(err, "Error creating manifest dir '%s'", manifestCacheDir)
		}
	}

	// acquire each kapp and cache it
	for _, kappObj := range manifest.ParsedKapps() {
		// build a directory path for the kapp in the manifest cache directory
		kappObj.SetCacheDir(cacheDir)

		log.Logger.Infof("Caching kapp '%s'", kappObj.FullyQualifiedId())
		log.Logger.Debugf("Kapp to cache: %#v", kappObj)

		// build a directory path for the kapp's .sugarkube cache directory
		sugarkubeCacheDir := getKappCachePath(kappObj.CacheDir())

		if _, err := os.Stat(sugarkubeCacheDir); err != nil {
			if os.IsNotExist(err) {
				log.Logger.Debugf("Creating kapp cache dir '%s'", sugarkubeCacheDir)
				err := os.MkdirAll(sugarkubeCacheDir, 0755)
				if err != nil {
					return errors.WithStack(err)
				}
			} else {
				return errors.Wrapf(err, "Error creating cache dir '%s'", sugarkubeCacheDir)
			}
		}

		acquirers, err := kappObj.Acquirers()
		if err != nil {
			return errors.WithStack(err)
		}

		err = acquireSource(manifest, acquirers, kappObj.CacheDir(), sugarkubeCacheDir, dryRun)
		if err != nil {
			return errors.WithStack(err)
		}
	}

	return nil
}

// Acquires each source and symlinks it to the target path in the cache directory.
// Runs all acquirers in parallel.
func acquireSource(manifest kapp.Manifest, acquirers []acquirer.Acquirer, rootDir string,
	cacheDir string, dryRun bool) error {
	doneCh := make(chan bool)
	errCh := make(chan error)

	log.Logger.Infof("Acquiring sources for manifest '%s'", manifest.Id())

	for _, acquirerImpl := range acquirers {
		go func(a acquirer.Acquirer) {
			acquirerId, err := a.FullyQualifiedId()
			if err != nil {
				errCh <- errors.Wrap(err, "Invalid acquirer ID")
				return
			}

			// todo - the no-op file acquirer doesn't actually cache files, so we need some object whose job it is
			// to create cache paths per-acquirer (or a method on each acquirer type)
			sourceDest := filepath.Join(cacheDir, acquirerId)

			if dryRun {
				log.Logger.Debugf("Dry run: Would acquire source into '%s'", sourceDest)
			} else {
				err := acquirer.Acquire(a, sourceDest)
				if err != nil {
					errCh <- errors.WithStack(err)
					return
				}
			}

			sourcePath := filepath.Join(sourceDest, a.Path())
			sourcePath = strings.TrimPrefix(sourcePath, rootDir)
			sourcePath = strings.TrimPrefix(sourcePath, "/")

			symLinkTarget := filepath.Join(rootDir, a.Id())

			var symLinksExist bool

			if _, err := os.Stat(symLinkTarget); err != nil {
				if os.IsNotExist(err) {
					log.Logger.Debugf("Symlinks don't exist at '%s'. Will create...", symLinkTarget)
					symLinksExist = false
				} else {
					errCh <- errors.WithStack(err)
					return
				}
			} else {
				log.Logger.Debugf("Symlinks already exist at '%s'", symLinkTarget)
				symLinksExist = true
			}

			if !symLinksExist {
				if dryRun {
					log.Logger.Debugf("Dry run. Would symlink cached source %s to %s", sourcePath, symLinkTarget)
				} else {
					if _, err := os.Stat(filepath.Join(rootDir, sourcePath)); err != nil {
						errCh <- errors.Wrapf(err, "Symlink source '%s' doesn't exist", sourcePath)
					}

					log.Logger.Debugf("Symlinking cached source %s to %s", sourcePath, symLinkTarget)
					err := os.Symlink(sourcePath, symLinkTarget)
					if err != nil {
						errCh <- errors.Wrapf(err, "Error symlinking source")
					}
				}
			}

			doneCh <- true
		}(acquirerImpl)
	}

	for success := 0; success < len(acquirers); success++ {
		select {
		case err := <-errCh:
			close(doneCh)
			log.Logger.Warnf("Error in acquirer goroutines: %s", err)
			return errors.Wrapf(err, "Error running acquirer in goroutine "+
				"for manifest '%s'", manifest.Id())
		case <-doneCh:
			log.Logger.Infof("%d acquirer(s) successfully completed for manifest '%s'",
				success+1, manifest.Id())
		}
	}

	log.Logger.Infof("Finished acquiring sources for manifest '%s'", manifest.Id())

	return nil
}

// Diffs a set of manifests against a cache directory and reports any differences
//func DiffCache(manifests []kapp.Manifest, cacheDir string) (???, error) {
// todo - implement
//}
