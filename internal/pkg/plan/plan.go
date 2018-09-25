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

package plan

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/sugarkube/sugarkube/internal/pkg/cacher"
	"github.com/sugarkube/sugarkube/internal/pkg/installer"
	"github.com/sugarkube/sugarkube/internal/pkg/kapp"
	"github.com/sugarkube/sugarkube/internal/pkg/log"
	"github.com/sugarkube/sugarkube/internal/pkg/provider"
	"os"
)

type Tranche struct {
	// The manifest associated with this tranche
	manifest kapp.Manifest
	// Kapps to install into the target cluster
	installables []kapp.Kapp
	// Kapps to destroy from the target cluster
	destroyables []kapp.Kapp
	// Kapps that are already in the target cluster so can be ignored
	ignorables []kapp.Kapp
}

type Plan struct {
	// installation/destruction phases. Tranches will be run sequentially, but
	// each kapp in the tranche will be processed in parallel
	tranche []Tranche
	// contains details of the target cluster
	stackConfig *kapp.StackConfig
	// a cache dir to run the (make) installer over. It should already have
	// been validated to match the stack config.
	cacheDir string
}

// A job to be run by a worker
type job struct {
	kappObj          kapp.Kapp
	stackConfig      *kapp.StackConfig
	manifestCacheDir string
	install          bool
	providerImpl     provider.Provider
	approved         bool
	dryRun           bool
}

// create a plan containing all kapps in the stackConfig, then filter out the
// ones that don't need running based on the current state of the target cluster
// as described by SOTs
func Create(stackConfig *kapp.StackConfig, cacheDir string) (*Plan, error) {

	tranches := make([]Tranche, 0)

	for _, manifest := range stackConfig.Manifests {
		installables := make([]kapp.Kapp, 0)
		destroyables := make([]kapp.Kapp, 0)

		for _, manifestKapp := range manifest.Kapps {
			if manifestKapp.ShouldBePresent {
				installables = append(installables, manifestKapp)
			} else {
				destroyables = append(destroyables, manifestKapp)
			}
		}

		tranche := Tranche{
			manifest:     manifest,
			installables: installables,
			destroyables: destroyables,
		}

		tranches = append(tranches, tranche)
	}

	plan := Plan{
		tranche:     tranches,
		stackConfig: stackConfig,
		cacheDir:    cacheDir,
	}

	// todo - use Sources of Truth (SOTs) to discover the current set of kapps installed
	// todo - diff the cluster state with the desired state from the manifests to
	// create a cluster diff

	return &plan, nil
}

// Run a plan to make a target cluster have the necessary kapps installed/
// destroyed to match the input manifests. Each tranche is run sequentially,
// and each kapp in each tranche is processed in parallel.
func (p *Plan) Run(approved bool, providerImpl provider.Provider, dryRun bool) error {

	if p.tranche == nil {
		log.Logger.Info("No tranches in plan to process")
		return nil
	}

	doneCh := make(chan bool)
	errCh := make(chan error)

	log.Logger.Debugf("Applying plan: %#v", p)

	for i, tranche := range p.tranche {
		manifestCacheDir := cacher.GetManifestCachePath(p.cacheDir, tranche.manifest)

		numWorkers := tranche.manifest.Options.Parallelisation
		if numWorkers == 0 {
			numWorkers = uint16(len(tranche.installables) + len(tranche.destroyables))
		}

		jobs := make(chan job, 100)

		// create the worker pool
		for w := uint16(0); w < numWorkers; w++ {
			go processKapp(jobs, doneCh, errCh)
		}

		for _, trancheKapp := range tranche.installables {
			install := true

			job := job{
				approved:         approved,
				dryRun:           dryRun,
				install:          install,
				kappObj:          trancheKapp,
				manifestCacheDir: manifestCacheDir,
				providerImpl:     providerImpl,
				stackConfig:      p.stackConfig,
			}

			jobs <- job
		}

		for _, trancheKapp := range tranche.destroyables {
			install := false

			job := job{
				approved:         approved,
				dryRun:           dryRun,
				install:          install,
				kappObj:          trancheKapp,
				manifestCacheDir: manifestCacheDir,
				providerImpl:     providerImpl,
				stackConfig:      p.stackConfig,
			}

			jobs <- job
		}

		// close the jobs channel so workers don't block waiting for any more
		close(jobs)

		totalOperations := len(tranche.installables) + len(tranche.destroyables)

		for success := 0; success < totalOperations; success++ {
			select {
			case err := <-errCh:
				log.Logger.Fatalf("Error processing kapp in tranche %d of plan: %s", i+1, err)
				close(doneCh)
				return errors.Wrapf(err, "Error processing kapp goroutine "+
					"in tranche %d of plan", i+1)
			case <-doneCh:
				log.Logger.Debugf("%d kapp(s) successfully processed in tranche %d",
					success+1, i+1)
			}
		}
	}

	log.Logger.Infof("Finished applying plan")

	return nil
}

// Installs or destroys a kapp using the appropriate Installer
func processKapp(jobs <-chan job, doneCh chan bool, errCh chan error) {

	for job := range jobs {
		kappObj := job.kappObj
		providerImpl := job.providerImpl
		stackConfig := job.stackConfig
		approved := job.approved
		dryRun := job.dryRun

		kappRootDir := cacher.GetKappRootPath(job.manifestCacheDir, kappObj)

		log.Logger.Debugf("Processing kapp '%s' in %s", kappObj.Id, kappRootDir)

		_, err := os.Stat(kappRootDir)
		if err != nil {
			msg := fmt.Sprintf("Kapp '%s' doesn't exist in the cache at '%s'",
				kappObj.Id, kappRootDir)
			log.Logger.Warn(msg)
			errCh <- errors.Wrap(err, msg)
		}

		kappObj.RootDir = kappRootDir

		// kapp exists, run the appropriate installer method
		installerImpl, err := installer.NewInstaller(installer.MAKE, providerImpl)
		if err != nil {
			errCh <- errors.Wrapf(err, "Error instantiating installer for "+
				"kapp '%s'", kappObj.Id)
		}

		// install the kapp
		if job.install {
			err := installer.Install(installerImpl, &kappObj, stackConfig, approved,
				providerImpl, dryRun)
			if err != nil {
				errCh <- errors.Wrapf(err, "Error installing kapp '%s'", kappObj.Id)
			}
		} else { // destroy the kapp
			err := installer.Destroy(installerImpl, &kappObj, stackConfig, approved,
				providerImpl, dryRun)
			if err != nil {
				errCh <- errors.Wrapf(err, "Error destroying kapp '%s'", kappObj.Id)
			}
		}

		doneCh <- true
	}
}
