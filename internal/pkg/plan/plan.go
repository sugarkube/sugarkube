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
	"github.com/sugarkube/sugarkube/internal/pkg/cmd/cli/cluster"
	"github.com/sugarkube/sugarkube/internal/pkg/constants"
	"github.com/sugarkube/sugarkube/internal/pkg/installer"
	"github.com/sugarkube/sugarkube/internal/pkg/kapp"
	"github.com/sugarkube/sugarkube/internal/pkg/log"
	"github.com/sugarkube/sugarkube/internal/pkg/provider"
	"os"
)

type task struct {
	action string
	kapp   kapp.Kapp
}

type tranche struct {
	// The manifest associated with this tranche
	manifest kapp.Manifest
	// tasks to run for this tranche (by default they'll run in parallel)
	tasks []task
}

type Plan struct {
	// installation/destruction phases. Tranches will be run sequentially, but
	// each kapp in the tranche will be processed in parallel
	tranche []tranche
	// contains details of the target cluster
	stackConfig *kapp.StackConfig
	// a cache dir to run the (make) installer over. It should already have
	// been validated to match the stack config.
	cacheDir string
	// whether to write templates for a kapp immediately before applying the kapp
	renderTemplates bool
}

// A job to be run by a worker
type job struct {
	task             task
	stackConfig      *kapp.StackConfig
	manifestCacheDir string
	renderTemplates  bool
	approved         bool
	dryRun           bool
}

// create a plan containing all kapps in the stackConfig, then filter out the
// ones that don't need running based on the current state of the target cluster
// as described by SOTs
// todo - add a 'runDirection' parameter to determine if we're spinning up the cluster or tearing it down
func Create(stackConfig *kapp.StackConfig, manifests []*kapp.Manifest, cacheDir string, includeSelector []string,
	excludeSelector []string, renderTemplates bool) (*Plan, error) {

	// selected kapps will be returned in the order in which they appear in manifests, not the order they're specified
	// in selectors
	selectedKapps, err := kapp.SelectKapps(manifests, includeSelector, excludeSelector)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	tranches := make([]tranche, 0)
	tasks := make([]task, 0)
	var previousManifest *kapp.Manifest

	for _, kappObj := range selectedKapps {
		var installDestroyTask *task

		if kappObj.State == kapp.PRESENT_KEY {
			installDestroyTask = &task{
				kapp:   kappObj,
				action: constants.TASK_ACTION_INSTALL,
			}
		} else if kappObj.State == kapp.ABSENT_KEY {
			installDestroyTask = &task{
				kapp:   kappObj,
				action: constants.TASK_ACTION_DESTROY,
			}
		}

		if installDestroyTask != nil {
			log.Logger.Debugf("Adding %s task for kapp '%s'", installDestroyTask.action,
				kappObj.FullyQualifiedId())
			tasks = append(tasks, *installDestroyTask)
		}

		if len(kappObj.PostActions) > 0 {
			for _, postAction := range kappObj.PostActions {
				if postAction == constants.TASK_ACTION_CLUSTER_UPDATE {
					actionTask := task{
						kapp:   kappObj,
						action: constants.TASK_ACTION_CLUSTER_UPDATE,
					}

					log.Logger.Debugf("Adding %s task for kapp '%s'", constants.TASK_ACTION_CLUSTER_UPDATE,
						kappObj.FullyQualifiedId())
					tasks = append(tasks, actionTask)
				}
			}
		}

		if previousManifest != nil && previousManifest.Id() != kappObj.GetManifest().Id() {
			tranche := tranche{
				manifest: *kappObj.GetManifest(),
				tasks:    tasks,
			}

			tranches = append(tranches, tranche)
			tasks = make([]task, 0)
		}

		previousManifest = kappObj.GetManifest()
	}

	if len(tasks) > 0 {
		tranche := tranche{
			manifest: *previousManifest,
			tasks:    tasks,
		}

		tranches = append(tranches, tranche)
	}

	plan := Plan{
		tranche:         tranches,
		stackConfig:     stackConfig,
		cacheDir:        cacheDir,
		renderTemplates: renderTemplates,
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
			numWorkers = uint16(len(tranche.tasks))
		}

		jobs := make(chan job, 10000)

		// create the worker pool
		for w := uint16(0); w < numWorkers; w++ {
			go processKapp(jobs, doneCh, errCh)
		}

		for _, task := range tranche.tasks {
			task.kapp.SetCacheDir(p.cacheDir)

			job := job{
				approved:        approved,
				dryRun:          dryRun,
				task:            task,
				stackConfig:     p.stackConfig,
				renderTemplates: p.renderTemplates,
			}

			jobs <- job
		}

		// close the jobs channel so workers don't block waiting for any more
		close(jobs)

		totalOperations := len(tranche.tasks)

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
		task := job.task
		kappObj := task.kapp
		stackConfig := job.stackConfig
		approved := job.approved
		dryRun := job.dryRun
		renderTemplates := job.renderTemplates

		kappRootDir := kappObj.CacheDir()
		log.Logger.Infof("Processing kapp '%s' in %s", kappObj.FullyQualifiedId(), kappRootDir)

		_, err := os.Stat(kappRootDir)
		if err != nil {
			msg := fmt.Sprintf("Kapp '%s' doesn't exist in the cache at '%s'", kappObj.Id, kappRootDir)
			log.Logger.Warn(msg)
			errCh <- errors.Wrap(err, msg)
		}

		providerImpl, err := provider.NewProvider(stackConfig)
		if err != nil {
			errCh <- errors.WithStack(err)
		}

		// kapp exists, Instantiate an installer in case we need it (for now, this will always be a Make installer)
		installerImpl, err := installer.NewInstaller(installer.MAKE, providerImpl)
		if err != nil {
			errCh <- errors.Wrapf(err, "Error instantiating installer for "+
				"kapp '%s'", kappObj.Id)
		}

		switch task.action {
		case constants.TASK_ACTION_INSTALL:
			err := installer.Install(installerImpl, &kappObj, stackConfig, approved, renderTemplates, dryRun)
			if err != nil {
				errCh <- errors.Wrapf(err, "Error installing kapp '%s'", kappObj.Id)
			}
			break
		case constants.TASK_ACTION_DESTROY:
			err := installer.Destroy(installerImpl, &kappObj, stackConfig, approved, renderTemplates, dryRun)
			if err != nil {
				errCh <- errors.Wrapf(err, "Error destroying kapp '%s'", kappObj.Id)
			}
			break
		case constants.TASK_ACTION_CLUSTER_UPDATE:
			err := cluster.UpdateCluster(os.Stdout, stackConfig, dryRun)
			if err != nil {
				errCh <- errors.Wrapf(err, "Error updating cluster, triggered by kapp '%s'", kappObj.Id)
			}
			break
		}

		doneCh <- true
	}
}
