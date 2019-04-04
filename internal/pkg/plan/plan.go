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
	"github.com/sugarkube/sugarkube/internal/pkg/interfaces"
	"github.com/sugarkube/sugarkube/internal/pkg/log"
	"os"
	"sort"
	"strings"
)

type task struct {
	action         string
	installableObj interfaces.IInstallable
}

type tranche struct {
	// The manifest associated with this tranche
	manifest interfaces.IManifest
	// tasks to run for this tranche (by default they'll run in parallel)
	tasks []task
}

type Plan struct {
	// installation/destruction phases. Tranches will be run sequentially, but
	// each kapp in the tranche will be processed in parallel
	tranches []tranche
	// contains details of the target cluster
	stack interfaces.IStack
	// whether to write templates for a kapp immediately before applying the kapp
	renderTemplates bool
}

// A job to be run by a worker
type job struct {
	task            task
	stack           interfaces.IStack
	renderTemplates bool
	approved        bool
	dryRun          bool
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
func Create(forward bool, stackObj interfaces.IStack, manifests []interfaces.IManifest,
	selectedInstallables []interfaces.IInstallable, renderTemplates bool, runPostActions bool) (*Plan, error) {

	tranches := make([]tranche, 0)
	tasks := make([]task, 0)
	var previousManifest interfaces.IManifest

	for _, installableObj := range selectedInstallables {
		var installDeleteTask *task

		// create a new tranche for each new manifest
		if previousManifest != nil && previousManifest.Id() != installableObj.ManifestId() &&
			len(tasks) > 0 {
			tranche := tranche{
				manifest: previousManifest,
				tasks:    tasks,
			}

			tranches = append(tranches, tranche)
			tasks = make([]task, 0)
			previousManifest = GetManifestById(manifests, installableObj.ManifestId())
		}

		// when creating a forward plan, we can install kapps...
		if forward && installableObj.State() == constants.PresentKey {
			installDeleteTask = &task{
				installableObj: installableObj,
				action:         constants.TaskActionInstall,
			}
			// but reverse plans must always delete them
		} else if !forward || installableObj.State() == constants.AbsentKey {
			installDeleteTask = &task{
				installableObj: installableObj,
				action:         constants.TaskActionDelete,
			}
		}

		if installDeleteTask != nil {
			log.Logger.Debugf("Adding %s task for kapp '%s'", installDeleteTask.action,
				installableObj.FullyQualifiedId())
			tasks = append(tasks, *installDeleteTask)
		}

		// when tearing down a cluster users may not want to execute any post-actions
		if len(installableObj.PostActions()) > 0 && runPostActions {
			for _, postAction := range installableObj.PostActions() {
				var actionTask *task
				if postAction == constants.TaskActionClusterUpdate {
					actionTask = &task{
						installableObj: installableObj,
						action:         constants.TaskActionClusterUpdate,
					}
				} else {
					log.Logger.Errorf("Invalid post_action encountered: %s", postAction)
				}

				// post action tasks are added to their own tranche to avoid race conditions
				if actionTask != nil {
					log.Logger.Debugf("Kapp '%s' has a post action task to add: %#v",
						installableObj.FullyQualifiedId(), actionTask)
					// add any previously queued tasks to a tranche
					if len(tasks) > 0 {
						tranche := tranche{
							manifest: GetManifestById(manifests, installableObj.ManifestId()),
							tasks:    tasks,
						}

						tranches = append(tranches, tranche)

						// reset the tasks list for the next iteration
						tasks = make([]task, 0)
					}

					log.Logger.Debugf("Adding %s task for kapp '%s'",
						actionTask.action, installableObj.FullyQualifiedId())

					// create a tranche just for the post action
					tranche := tranche{
						manifest: GetManifestById(manifests, installableObj.ManifestId()),
						tasks:    []task{*actionTask},
					}

					tranches = append(tranches, tranche)
				}
			}
		}

		if len(tasks) > 0 && previousManifest != nil &&
			previousManifest.Id() != installableObj.ManifestId() {
			tranche := tranche{
				manifest: previousManifest,
				tasks:    tasks,
			}

			tranches = append(tranches, tranche)
			tasks = make([]task, 0)
		}

		previousManifest = GetManifestById(manifests, installableObj.ManifestId())
	}

	// add a tranche if there are any tasks for the final manifest
	if len(tasks) > 0 {
		tranche := tranche{
			manifest: previousManifest,
			tasks:    tasks,
		}

		tranches = append(tranches, tranche)
	}

	plan := Plan{
		tranches:        tranches,
		stack:           stackObj,
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

// Returns a manifest by ID. Panics if it doesn't exist (for a simpler interface)
func GetManifestById(manifests []interfaces.IManifest, manifestId string) interfaces.IManifest {
	for _, manifest := range manifests {
		if manifest.Id() == manifestId {
			return manifest
		}
	}

	panic(fmt.Sprintf("No manifest found with ID '%s'", manifestId))
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
// deleted to match the input manifests. Each tranche is run sequentially,
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

	var previousManifestId string

	for i, tranche := range p.tranches {

		numWorkers := tranche.manifest.Parallelisation()
		if numWorkers == 0 {
			numWorkers = uint16(len(tranche.tasks))
		}

		jobs := make(chan job, 10000)

		// create the worker pool
		for w := uint16(0); w < numWorkers; w++ {
			go processKapp(jobs, doneCh, errCh)
		}

		for _, task := range tranche.tasks {
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
				return errors.Wrapf(err, "Error processing kapp goroutine "+
					"in tranche %d of plan", i+1)
			case <-doneCh:
				log.Logger.Infof("%d kapp(s) successfully processed in tranche %d",
					success+1, i+1)
			}
		}

		// refresh the provider vars after each tranche
		err := p.stack.RefreshProviderVars()
		if err != nil {
			return errors.WithStack(err)
		}

		// make sure we don't clean up the registry if we're just running multiple tranches for a
		// manifest (e.g. because post-actions are being executed in their own tranche)
		if previousManifestId != "" && previousManifestId != tranche.manifest.Id() {
			// clean up the registry if the manifest ID has changed
			deleteNonFullyQualifiedOutputs(p.stack.GetRegistry())
		}

		previousManifestId = tranche.manifest.Id()
	}

	// clean up the registry after applying the plan to catch stragglers from the final tranche
	deleteNonFullyQualifiedOutputs(p.stack.GetRegistry())

	log.Logger.Infof("Finished applying plan")

	return nil
}

// Installs or deletes a kapp using the appropriate Installer
func processKapp(jobs <-chan job, doneCh chan bool, errCh chan error) {

	for job := range jobs {
		task := job.task
		installableObj := task.installableObj
		stackObj := job.stack
		approved := job.approved
		dryRun := job.dryRun
		renderTemplates := job.renderTemplates

		kappRootDir := installableObj.GetCacheDir()
		log.Logger.Infof("Processing kapp '%s' in %s", installableObj.FullyQualifiedId(), kappRootDir)

		// todo - print (to stdout) detais of the kapp being executed

		_, err := os.Stat(kappRootDir)
		if err != nil {
			msg := fmt.Sprintf("Kapp '%s' doesn't exist in the cache at '%s'", installableObj.Id(), kappRootDir)
			log.Logger.Warn(msg)
			errCh <- errors.Wrap(err, msg)
		}

		// kapp exists, Instantiate an installer in case we need it (for now, this will always be a Make installer)
		installerImpl, err := installer.New(installer.MAKE, stackObj.GetProvider())
		if err != nil {
			errCh <- errors.Wrapf(err, "Error instantiating installer for "+
				"kapp '%s'", installableObj.Id())
		}

		switch task.action {
		case constants.TaskActionInstall:
			err := installerImpl.Install(installableObj, stackObj, approved, renderTemplates, dryRun)
			if err != nil {
				errCh <- errors.Wrapf(err, "Error installing kapp '%s'", installableObj.Id())
			}

			err = addOutputsToRegistry(installableObj, stackObj.GetRegistry())
			if err != nil {
				errCh <- errors.WithStack(err)
			}
			break
		case constants.TaskActionDelete:
			err := installerImpl.Delete(installableObj, stackObj, approved, renderTemplates, dryRun)
			if err != nil {
				errCh <- errors.Wrapf(err, "Error deleting kapp '%s'", installableObj.Id())
			}

			err = addOutputsToRegistry(installableObj, stackObj.GetRegistry())
			if err != nil {
				errCh <- errors.WithStack(err)
			}
			break
		case constants.TaskActionClusterUpdate:
			if approved {
				log.Logger.Info("Running cluster update action")
				err := cluster.UpdateCluster(os.Stdout, stackObj, true, dryRun)
				if err != nil {
					errCh <- errors.Wrapf(err, "Error updating cluster, triggered by kapp '%s'",
						installableObj.Id())
				}
			} else {
				log.Logger.Info("Skipping cluster update action since the approved=false")
			}
			break
		}

		doneCh <- true
	}
}

// Adds output from an installable to the registry
func addOutputsToRegistry(installableObj interfaces.IInstallable, registry interfaces.IRegistry) error {
	outputs, err := installableObj.GetOutputs()
	if err != nil {
		return errors.Wrapf(err, "Error getting output for kapp '%s'", installableObj.Id())
	}

	// data under the short key can be used by other kapps in the manifest
	prefix := strings.Join([]string{constants.RegistryKeyOutputs, installableObj.Id()}, constants.RegistryFieldSeparator)
	// kapps in different manifests need to use the fully qualified ID
	fullyQualifiedPrefix := strings.Join([]string{constants.RegistryKeyOutputs,
		installableObj.FullyQualifiedId()}, constants.RegistryFieldSeparator)

	// store the output under both the kapp's fully-qualified ID and its short, inter-manifest kapp
	for key, output := range outputs {
		fullKey := strings.Join([]string{prefix, key}, constants.RegistryFieldSeparator)
		err = registry.Set(fullKey, output)
		if err != nil {
			return errors.WithStack(err)
		}

		fullyQualifiedFullKey := strings.Join([]string{fullyQualifiedPrefix, key}, constants.RegistryFieldSeparator)
		err = registry.Set(fullyQualifiedFullKey, output)
		if err != nil {
			return errors.WithStack(err)
		}
	}

	return nil
}

// Deletes all outputs from the registry that aren't fully qualified
func deleteNonFullyQualifiedOutputs(registry interfaces.IRegistry) {
	outputs, ok := registry.Get(constants.RegistryKeyOutputs)
	if !ok {
		return
	}

	// iterate through all the keys for those that aren't fully qualified and delete them
	for k, _ := range outputs.(map[string]interface{}) {
		if !strings.Contains(k, constants.NamespaceSeparator) {
			fullKey := strings.Join([]string{
				constants.RegistryKeyOutputs, k}, constants.RegistryFieldSeparator)
			registry.Delete(fullKey)
		}
	}
}
