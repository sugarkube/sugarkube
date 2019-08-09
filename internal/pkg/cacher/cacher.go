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
	"github.com/sugarkube/sugarkube/internal/pkg/interfaces"
	"github.com/sugarkube/sugarkube/internal/pkg/log"
	"os"
	"path/filepath"
	"strings"
)

const CacheDir = ".sugarkube"

type Ider interface {
	Id() string
}

type CacheGrouper interface {
	Ider
	Installables() []interfaces.IInstallable
}

// Cache a group of cacheable objects under a root directory
func CacheManifest(cacheGroup CacheGrouper, rootCacheDir string, dryRun bool) error {

	// create a directory to cache all kapps in this cacheGroup in
	groupCacheDir := filepath.Join(rootCacheDir, cacheGroup.Id())

	err := createDirectoryIfMissing(groupCacheDir)
	if err != nil {
		return errors.WithStack(err)
	}

	// acquire each kapp and cache it
	for _, installableObj := range cacheGroup.Installables() {
		log.Logger.Infof("Caching kapp '%s'", installableObj.FullyQualifiedId())
		log.Logger.Debugf("Kapp to cache: %#v", installableObj)

		err := installableObj.SetWorkspaceDir(rootCacheDir)
		if err != nil {
			return errors.WithStack(err)
		}

		acquirers, err := installableObj.Acquirers()
		if err != nil {
			return errors.WithStack(err)
		}

		err = acquireSources(cacheGroup.Id(), acquirers, installableObj.GetCacheDir(),
			dryRun)
		if err != nil {
			return errors.WithStack(err)
		}
	}

	return nil
}

// Acquires each source and symlinks it to the target path in the cache directory.
// Runs all acquirers in parallel.
func acquireSources(manifestId string, acquirers map[string]acquirer.Acquirer, kappTopLevelCacheDir string,
	dryRun bool) error {

	// build a directory path for the kapp's .sugarkube cache directory
	kappHiddenCacheDir := filepath.Join(kappTopLevelCacheDir, CacheDir)

	err := createDirectoryIfMissing(kappHiddenCacheDir)
	if err != nil {
		return errors.WithStack(err)
	}

	doneCh := make(chan bool)
	errCh := make(chan error)

	log.Logger.Infof("Acquiring sources for manifest '%s'", manifestId)

	for _, acquirerImpl := range acquirers {
		go func(a acquirer.Acquirer) {
			acquirerId, err := a.FullyQualifiedId()
			if err != nil {
				errCh <- errors.Wrap(err, "Invalid acquirer ID")
				return
			}

			// todo - the no-op file acquirer doesn't actually cache files, so we need some object whose job it is
			// to create cache paths per-acquirer (or a method on each acquirer type)
			sourceDest := filepath.Join(kappHiddenCacheDir, acquirerId)

			if dryRun {
				log.Logger.Debugf("Dry run: Would acquire source into '%s'", sourceDest)
			} else {
				err := acquirer.Acquire(a, sourceDest)
				if err != nil {
					errCh <- errors.WithStack(err)
					return
				}
			}

			// todo - fix creating symlinks when the path is just '/'
			sourcePath := filepath.Join(sourceDest, a.Path())
			sourcePath = strings.TrimPrefix(sourcePath, kappTopLevelCacheDir)
			sourcePath = strings.TrimPrefix(sourcePath, "/")

			var symLinkTarget string
			if a.Id() != "" {
				symLinkTarget = filepath.Join(kappTopLevelCacheDir, a.Id())
			} else {
				fqId, err := a.FullyQualifiedId()
				if err != nil {
					errCh <- errors.WithStack(err)
					return
				}
				symLinkTarget = filepath.Join(kappTopLevelCacheDir, fqId)
			}

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
					if _, err := os.Stat(filepath.Join(kappTopLevelCacheDir, sourcePath)); err != nil {
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
				"for manifest '%s'", manifestId)
		case <-doneCh:
			log.Logger.Infof("%d acquirer(s) successfully completed for manifest '%s'",
				success+1, manifestId)
		}
	}

	log.Logger.Infof("Finished acquiring sources for manifest '%s'", manifestId)

	return nil
}

// Creates a directory if it doesn't exist
func createDirectoryIfMissing(path string) error {
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			log.Logger.Debugf("Creating dir '%s'", path)
			err := os.MkdirAll(path, 0755)
			if err != nil {
				return errors.WithStack(err)
			}
		} else {
			return errors.Wrapf(err, "Error creating dir '%s'", path)
		}
	}

	return nil
}

// Diffs a set of manifests against a cache directory and reports any differences
//func DiffCache(manifests []kapp.Manifest, cacheDir string) (???, error) {
// todo - implement
//}
