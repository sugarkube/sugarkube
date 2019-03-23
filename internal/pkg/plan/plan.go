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
	"github.com/sirupsen/logrus"
	"github.com/sugarkube/sugarkube/internal/pkg/cmd/cli/cluster"
	"github.com/sugarkube/sugarkube/internal/pkg/constants"
	"github.com/sugarkube/sugarkube/internal/pkg/installer"
	"github.com/sugarkube/sugarkube/internal/pkg/kapp"
	"github.com/sugarkube/sugarkube/internal/pkg/log"
	"github.com/sugarkube/sugarkube/internal/pkg/stack"
	"os"
	"sort"
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
	tranches []tranche
	// contains details of the target cluster
	stack *stack.Stack
	// a cache dir to run the (make) installer over. It should already have
	// been validated to match the stack config.
	cacheDir string
	// whether to write templates for a kapp immediately before applying the kapp
	renderTemplates bool
}

// A job to be run by a worker
type job struct {
	task             task
	stack            *stack.Stack
	manifestCacheDir string
	renderTemplates  bool
	approved         bool
	dryRun           bool
}

// create a plan containing all kapps in the stack, then filter out the
// ones that don't need running based on the current state of the target cluster
// as described by SOTs (todo).
// If the `forward` parameter is true, the plan will be ordered so that kapps
// are applied from first to last in the orders specified in the manifests which
// will install kapps into a new cluster. If it's false, the order will be
// reversed which will be useful when tearing down a cluster.
// If the `runPostActions` parameter is false, no post actions will be executed.
// This can be useful to quickly tear down a cluster.
func Create(forward bool, stackObj *stack.Stack, manifests []*kapp.Manifest,
	cacheDir string, includeSelector []string, excludeSelector []string,
	renderTemplates bool, runPostActions bool) (*Plan, error) {

	// selected kapps will be returned in the order in which they appear in manifests, not the order they're specified
	// in selectors
	selectedKapps, err := kapp.SelectInstallables(manifests, includeSelector, excludeSelector)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	tranches := make([]tranche, 0)
	tasks := make([]task, 0)
	var previousManifest *kapp.Manifest

	for _, kappObj := range selectedKapps {
		var installDestroyTask *task

		// create a new tranche for each new manifest
		if previousManifest != nil && previousManifest.Id() != kappObj.GetManifest().Id() &&
			len(tasks) > 0 {
			tranche := tranche{
				manifest: *previousManifest,
				tasks:    tasks,
			}

			tranches = append(tranches, tranche)
			tasks = make([]task, 0)
			previousManifest = kappObj.GetManifest()
		}

		// when creating a forward plan, we can install kapps...
		if forward && kappObj.State == constants.PresentKey {
			installDestroyTask = &task{
				kapp:   kappObj,
				action: constants.TaskActionInstall,
			}
			// but reverse plans must always destroy them
		} else if !forward || kappObj.State == constants.AbsentKey {
			installDestroyTask = &task{
				kapp:   kappObj,
				action: constants.TaskActionDestroy,
			}
		}

		if installDestroyTask != nil {
			log.Logger.Debugf("Adding %s task for kapp '%s'", installDestroyTask.action,
				kappObj.FullyQualifiedId())
			tasks = append(tasks, *installDestroyTask)
		}

		// when tearing down a cluster users may not want to execute any post-actions
		if len(kappObj.PostActions) > 0 && runPostActions {
			for _, postAction := range kappObj.PostActions {
				var actionTask *task
				if postAction == constants.TaskActionClusterUpdate {
					actionTask = &task{
						kapp:   kappObj,
						action: constants.TaskActionClusterUpdate,
					}
				} else {
					log.Logger.Errorf("Invalid post_action encountered: %s", postAction)
				}

				// post action tasks are added to their own tranche to avoid race conditions
				if actionTask != nil {
					log.Logger.Debugf("Kapp '%s' has a post action task to add: %#v",
						kappObj.FullyQualifiedId(), actionTask)
					// add any previously queued tasks to a tranche
					if len(tasks) > 0 {
						tranche := tranche{
							manifest: *kappObj.GetManifest(),
							tasks:    tasks,
						}

						tranches = append(tranches, tranche)

						// reset the tasks list for the next iteration
						tasks = make([]task, 0)
					}

					log.Logger.Debugf("Adding %s task for kapp '%s'",
						actionTask.action, kappObj.FullyQualifiedId())

					// create a tranche just for the post action
					tranche := tranche{
						manifest: *kappObj.GetManifest(),
						tasks:    []task{*actionTask},
					}

					tranches = append(tranches, tranche)
				}
			}
		}

		if len(tasks) > 0 && previousManifest != nil &&
			previousManifest.Id() != kappObj.GetManifest().Id() {
			tranche := tranche{
				manifest: *previousManifest,
				tasks:    tasks,
			}

			tranches = append(tranches, tranche)
			tasks = make([]task, 0)
		}

		previousManifest = kappObj.GetManifest()
	}

	// add a tranche if there are any tasks for the final manifest
	if len(tasks) > 0 {
		tranche := tranche{
			manifest: *previousManifest,
			tasks:    tasks,
		}

		tranches = append(tranches, tranche)
	}

	plan := Plan{
		tranches:        tranches,
		stack:           stackObj,
		cacheDir:        cacheDir,
		renderTemplates: renderTemplates,
	}

	if !forward {
		log.Logger.Debugf("Reversing plan...")
		reverse(&plan)
	}

	// todo - use Sources of Truth (SOTs) to discover the current set of kapps installed
	// todo - diff the cluster state with the desired state from the manifests to
	// create a cluster diff

	return &plan, nil
}

// Reverses the order of all tranches and all tasks within each tranche to reverse
// the run order of the plan
func reverse(plan *Plan) {
	// reverse the tranches
	sort.SliceStable(plan.tranches, func(i, j int) bool {
		return true
	})

	// reverse the tasks within each tranche
	for i := range plan.tranches {
		sort.SliceStable(plan.tranches[i].tasks, func(i, j int) bool {
			return true
		})
	}
}

// Run a plan to make a target cluster have the necessary kapps installed/
// destroyed to match the input manifests. Each tranche is run sequentially,
// and each kapp in each tranche is processed in parallel.
func (p *Plan) Run(approved bool, dryRun bool) error {

	if p.tranches == nil {
		log.Logger.Info("No tranches in plan to process")
		return nil
	}

	doneCh := make(chan bool)
	errCh := make(chan error)

	log.Logger.Info("Applying plan...")
	log.Logger.Debugf("Applying plan: %#v", p)

	for i, tranche := range p.tranches {
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
				stack:           p.stack,
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
				if log.Logger.Level == logrus.DebugLevel {
					log.Logger.Fatalf("Error processing kapp in tranche %d of plan: %+v", i+1, err)
				} else {
					log.Logger.Fatalf("Error processing kapp in tranche %d of plan: %v\n"+
						"Run with `-l debug` for a full stacktrace.", i+1, err)
				}
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
		stackObj := job.stack
		approved := job.approved
		dryRun := job.dryRun
		renderTemplates := job.renderTemplates

		kappRootDir := kappObj.CacheDir()
		log.Logger.Infof("Processing kapp '%s' in %s", kappObj.FullyQualifiedId(), kappRootDir)

		// todo - print (to stdout) detais of the kapp being executed

		_, err := os.Stat(kappRootDir)
		if err != nil {
			msg := fmt.Sprintf("Kapp '%s' doesn't exist in the cache at '%s'", kappObj.Id, kappRootDir)
			log.Logger.Warn(msg)
			errCh <- errors.Wrap(err, msg)
		}

		// kapp exists, Instantiate an installer in case we need it (for now, this will always be a Make installer)
		installerImpl, err := installer.NewInstaller(installer.MAKE, stackObj.Provider)
		if err != nil {
			errCh <- errors.Wrapf(err, "Error instantiating installer for "+
				"kapp '%s'", kappObj.Id)
		}

		switch task.action {
		case constants.TaskActionInstall:
			err := installer.Install(installerImpl, &kappObj, stackObj, approved, renderTemplates, dryRun)
			if err != nil {
				errCh <- errors.Wrapf(err, "Error installing kapp '%s'", kappObj.Id)
			}
			break
		case constants.TaskActionDestroy:
			err := installer.Destroy(installerImpl, &kappObj, stackObj, approved, renderTemplates, dryRun)
			if err != nil {
				errCh <- errors.Wrapf(err, "Error destroying kapp '%s'", kappObj.Id)
			}
			break
		case constants.TaskActionClusterUpdate:
			if approved {
				log.Logger.Info("Running cluster update action")
				err := cluster.UpdateCluster(os.Stdout, stackObj, true, dryRun)
				if err != nil {
					errCh <- errors.Wrapf(err, "Error updating cluster, triggered by kapp '%s'", kappObj.Id)
				}
			} else {
				log.Logger.Info("Skipping cluster update action since the approved=false")
			}
			break
		}

		doneCh <- true
	}
}
