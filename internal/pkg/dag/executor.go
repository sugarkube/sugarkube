/*
 * Copyright 2019 The Sugarkube Authors
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

package dag

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/sugarkube/sugarkube/internal/pkg/cmd/cli/cluster"
	"github.com/sugarkube/sugarkube/internal/pkg/constants"
	"github.com/sugarkube/sugarkube/internal/pkg/installer"
	"github.com/sugarkube/sugarkube/internal/pkg/interfaces"
	"github.com/sugarkube/sugarkube/internal/pkg/log"
	"github.com/sugarkube/sugarkube/internal/pkg/structs"
	"os"
	"path/filepath"
)

// todo - make this configurable
const parallelisation = 5

// todo - should we add options to skip templating or running post actions?
// Traverses the DAG executing the named action on marked/processable nodes depending on the
// given options
func (d *dag) Execute(action string, stackObj interfaces.IStack, plan bool, apply bool, dryRun bool) error {
	processCh := make(chan NamedNode, parallelisation)
	doneCh := make(chan NamedNode)
	errCh := make(chan error)

	log.Logger.Infof("Executing DAG with plan=%v, apply=%v, dryRun=%v", plan, apply, dryRun)

	// create the worker pool
	for w := uint16(0); w < parallelisation; w++ {
		go worker(processCh, doneCh, errCh, action, stackObj, plan, apply, dryRun)
	}

	var finishedCh <-chan bool

	switch action {
	case constants.TaskActionInstall:
		finishedCh = d.WalkDown(processCh, doneCh)
		break
	// todo - implement
	//case constants.TaskActionDelete:
	//	finishedCh = d.WalkUp(processCh, doneCh)
	//	break
	default:
		return fmt.Errorf("Invalid action on DAG: %s", action)
	}

	for {
		select {
		case err := <-errCh:
			return errors.Wrapf(err, "Error processing kapp")
		case node := <-doneCh:
			if node.shouldProcess {
				log.Logger.Infof("Kapp '%s' processed", node.Name())
			}
		case <-finishedCh:
			log.Logger.Infof("Finished processing kapps")
			break
		}
	}
}

// Processes an installable, either installing/deleting it, running post actions or
// loading its outputs, etc.
func worker(processCh <-chan NamedNode, doneCh chan NamedNode, errCh chan error,
	action string, stackObj interfaces.IStack, plan bool, apply bool, dryRun bool) {

	for node := range processCh {
		installableObj := node.installableObj

		kappRootDir := installableObj.GetCacheDir()
		log.Logger.Infof("Processing kapp '%s' in %s", installableObj.FullyQualifiedId(), kappRootDir)

		// todo - print (to stdout) details of the kapp being executed

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

		// todo - support installing, deleting, templating and printing out the vars for each
		//  marked kapp
		switch action {
		case constants.TaskActionInstall:
			install(node, installerImpl, stackObj, plan, apply, dryRun, errCh)
			break
		case constants.TaskActionDelete:
			err := installerImpl.Delete(installableObj, stackObj, approved, true, dryRun)
			if err != nil {
				errCh <- errors.Wrapf(err, "Error deleting kapp '%s'", installableObj.Id())
			}

			// only add outputs if we've actually run the kapp
			if approved && installableObj.HasOutputs() {
				err := installerImpl.Output(installableObj, stackObj, approved, true, dryRun)
				if err != nil {
					errCh <- errors.Wrapf(err, "Error getting output for kapp '%s'", installableObj.Id())
				}

				// todo - add options to control whether to add outputs on installation (default), deletion or both
				err = addOutputsToRegistry(installableObj, stackObj.GetRegistry(), dryRun)
				if err != nil {
					errCh <- errors.WithStack(err)
				}

				// rerender templates so they can use kapp outputs (e.g. before adding the paths to rendered templates as provider vars)
				err = renderKappTemplates(stackObj, installableObj, dryRun)
				if err != nil {
					errCh <- errors.WithStack(err)
				}
			}
			break
		}

		doneCh <- node
	}
}

// Implements the install action. Nodes that should be processed are installed. All nodes load any outputs
// and merge them with their parents' outputs.
func install(node NamedNode, installerImpl interfaces.IInstaller, stackObj interfaces.IStack,
	plan bool, apply bool, dryRun bool, errCh chan error) {

	installableObj := node.installableObj

	// only plan or apply kapps that have been flagged for processing
	if node.shouldProcess && plan {
		err := installerImpl.Install(installableObj, stackObj, false, true, dryRun)
		if err != nil {
			errCh <- errors.Wrapf(err, "Error installing kapp '%s'", installableObj.Id())
		}
	}

	if node.shouldProcess && apply {
		err := installerImpl.Install(installableObj, stackObj, true, true, dryRun)
		if err != nil {
			errCh <- errors.Wrapf(err, "Error installing kapp '%s'", installableObj.Id())
		}
	}

	// try to load kapp outputs and fail if we can't
	if installableObj.HasOutputs() {
		err := installerImpl.Output(installableObj, stackObj, true, dryRun)
		if err != nil {
			errCh <- errors.Wrapf(err, "Error getting output for kapp '%s'", installableObj.Id())
		}

		// todo - merge the outputs with the parents' outputs and save as a property on the installable

		err = addOutputsToRegistry(installableObj, stackObj.GetRegistry(), dryRun)
		if err != nil {
			errCh <- errors.WithStack(err)
		}

		// rerender templates so they can use kapp outputs (e.g. before adding the paths to rendered templates as provider vars)
		err = renderKappTemplates(stackObj, installableObj, dryRun)
		if err != nil {
			errCh <- errors.WithStack(err)
		}
	}

	// execute any post actions if we've just actually installed the kapp.
	if node.shouldProcess && apply && len(installableObj.PostActions()) > 0 {
		for _, postAction := range installableObj.PostActions() {
			executePostAction(postAction, installableObj, stackObj, errCh, dryRun)
		}
	}
}

// Executes post actions
func executePostAction(postAction structs.PostAction, installableObj interfaces.IInstallable,
	stackObj interfaces.IStack, errCh chan error, dryRun bool) {
	switch postAction.Id {
	case constants.TaskActionClusterUpdate:
		log.Logger.Info("Running cluster update action")
		err := cluster.UpdateCluster(os.Stdout, stackObj, true, dryRun)
		if err != nil {
			errCh <- errors.Wrapf(err, "Error updating cluster, triggered by kapp '%s'",
				installableObj.Id())
		}
		break
	case constants.TaskAddProviderVarsFiles:
		log.Logger.Infof("Running action to add provider vars dirs")
		// todo - run each path through the templater
		for _, path := range postAction.Params {
			if !filepath.IsAbs(path) {
				// convert the relative path to absolute
				path = filepath.Join(installableObj.GetConfigFileDir(), path)
			}

			log.Logger.Debugf("Adding provider vars dir: %s", path)
			stackObj.GetProvider().AddVarsPath(path)
		}
		break
	}
}
