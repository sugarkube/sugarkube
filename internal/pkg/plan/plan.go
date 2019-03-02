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
	// whether to write templates for a kapp immediately before applying the kapp
	writeTemplates bool
}

// A job to be run by a worker
type job struct {
	kappObj          kapp.Kapp
	stackConfig      *kapp.StackConfig
	manifestCacheDir string
	writeTemplates   bool
	install          bool
	approved         bool
	dryRun           bool
}

// create a plan containing all kapps in the stackConfig, then filter out the
// ones that don't need running based on the current state of the target cluster
// as described by SOTs
func Create(stackConfig *kapp.StackConfig, manifests []*kapp.Manifest, cacheDir string, includeSelector []string,
	excludeSelector []string, writeTemplates bool) (*Plan, error) {

	tranches := make([]Tranche, 0)

	// ordering is significant when creating a plan, so we need to iterate over each manifest, select the required
	// kapps, then create a tranche
	for _, manifest := range manifests {
		installables := make([]kapp.Kapp, 0)
		destroyables := make([]kapp.Kapp, 0)

		log.Logger.Debugf("Selecting kapps from manifest '%s'", manifest.Id())

		selectedKapps, err := kapp.SelectKapps([]*kapp.Manifest{manifest}, includeSelector, excludeSelector)
		if err != nil {
			return nil, errors.WithStack(err)
		}

		for _, manifestKapp := range selectedKapps {
			if manifestKapp.State == kapp.PRESENT_KEY {
				installables = append(installables, manifestKapp)
			} else if manifestKapp.State == kapp.ABSENT_KEY {
				destroyables = append(destroyables, manifestKapp)
			} else {
				log.Logger.Warnf("State of kapp '%s' is '%s'. Use '%s' or '%s' to install or delete the kapp. "+
					"It will be ignored.", manifestKapp.FullyQualifiedId(), manifestKapp.State, kapp.PRESENT_KEY,
					kapp.ABSENT_KEY)
			}
		}

		if len(installables) > 0 || len(destroyables) > 0 {
			tranche := Tranche{
				manifest:     *manifest,
				installables: installables,
				destroyables: destroyables,
			}

			tranches = append(tranches, tranche)
		}
	}

	plan := Plan{
		tranche:        tranches,
		stackConfig:    stackConfig,
		cacheDir:       cacheDir,
		writeTemplates: writeTemplates,
	}

	// todo - use Sources of Truth (SOTs) to discover the current set of kapps installed
	// todo - diff the cluster state with the desired state from the manifests to
	// create a cluster diff

	return &plan, nil
}

// Run a plan to make a target cluster have the necessary kapps installed/
// destroyed to match the input manifests. Each tranche is run sequentially,
// and each kapp in each tranche is processed in parallel.
func (p *Plan) Run(approved bool, dryRun bool) error {

	if p.tranche == nil {
		log.Logger.Info("No tranches in plan to process")
		return nil
	}

	doneCh := make(chan bool)
	errCh := make(chan error)

	log.Logger.Info("Applying plan...")
	log.Logger.Debugf("Applying plan: %#v", p)

	for i, tranche := range p.tranche {
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
			trancheKapp.SetCacheDir(p.cacheDir)

			job := job{
				approved:       approved,
				dryRun:         dryRun,
				install:        install,
				kappObj:        trancheKapp,
				stackConfig:    p.stackConfig,
				writeTemplates: p.writeTemplates,
			}

			jobs <- job
		}

		for _, trancheKapp := range tranche.destroyables {
			install := false
			trancheKapp.SetCacheDir(p.cacheDir)

			job := job{
				approved:       approved,
				dryRun:         dryRun,
				install:        install,
				kappObj:        trancheKapp,
				stackConfig:    p.stackConfig,
				writeTemplates: p.writeTemplates,
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
				log.Logger.Infof("%d kapp(s) successfully processed in tranche %d",
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
		stackConfig := job.stackConfig
		approved := job.approved
		dryRun := job.dryRun
		writeTemplates := job.writeTemplates

		kappRootDir := kappObj.CacheDir()
		log.Logger.Infof("Processing kapp '%s' in %s", kappObj.FullyQualifiedId(),
			kappRootDir)

		_, err := os.Stat(kappRootDir)
		if err != nil {
			msg := fmt.Sprintf("Kapp '%s' doesn't exist in the cache at '%s'",
				kappObj.Id, kappRootDir)
			log.Logger.Warn(msg)
			errCh <- errors.Wrap(err, msg)
		}

		providerImpl, err := provider.NewProvider(stackConfig)
		if err != nil {
			errCh <- errors.WithStack(err)
		}

		// kapp exists, run the appropriate installer method (for now, this will
		// always be a Make installer)
		installerImpl, err := installer.NewInstaller(installer.MAKE, providerImpl)
		if err != nil {
			errCh <- errors.Wrapf(err, "Error instantiating installer for "+
				"kapp '%s'", kappObj.Id)
		}

		// install the kapp
		if job.install {
			err := installer.Install(installerImpl, &kappObj, stackConfig, approved, writeTemplates, dryRun)
			if err != nil {
				errCh <- errors.Wrapf(err, "Error installing kapp '%s'", kappObj.Id)
			}
		} else { // destroy the kapp
			err := installer.Destroy(installerImpl, &kappObj, stackConfig, approved, writeTemplates, dryRun)
			if err != nil {
				errCh <- errors.Wrapf(err, "Error destroying kapp '%s'", kappObj.Id)
			}
		}

		doneCh <- true
	}
}
