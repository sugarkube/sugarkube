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

package plan

import (
	"bytes"
	"fmt"
	"github.com/imdario/mergo"
	"github.com/mattn/go-shellwords"
	"github.com/pkg/errors"
	"github.com/sugarkube/sugarkube/internal/pkg/cmd/cli/cluster"
	"github.com/sugarkube/sugarkube/internal/pkg/config"
	"github.com/sugarkube/sugarkube/internal/pkg/constants"
	"github.com/sugarkube/sugarkube/internal/pkg/installer"
	"github.com/sugarkube/sugarkube/internal/pkg/interfaces"
	"github.com/sugarkube/sugarkube/internal/pkg/log"
	"github.com/sugarkube/sugarkube/internal/pkg/printer"
	"github.com/sugarkube/sugarkube/internal/pkg/registry"
	"github.com/sugarkube/sugarkube/internal/pkg/structs"
	"github.com/sugarkube/sugarkube/internal/pkg/utils"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

// Traverses the DAG executing the named action on marked/processable nodes depending on the
// given options
func (d *Dag) Execute(action string, stackObj interfaces.IStack, plan bool, approved bool, skipPreActions bool,
	skipPostActions bool, ignoreErrors bool, dryRun bool) error {
	numWorkers := config.CurrentConfig.NumWorkers

	processCh := make(chan NamedNode, numWorkers)
	doneCh := make(chan NamedNode)
	errCh := make(chan error)

	log.Logger.Infof("Executing DAG with action=%s, plan=%v, approved=%v, "+
		"skipPreActions=%v, skipPostActions=%v, ignoreErrors=%v, dryRun=%v", action, plan, approved, skipPreActions,
		skipPostActions, ignoreErrors, dryRun)

	_, err := printer.Fprintln("[yellow]Executing the DAG...")
	if err != nil {
		return errors.WithStack(err)
	}

	// create the worker pool
	for w := int(0); w < numWorkers; w++ {
		go worker(d, processCh, doneCh, errCh, action, stackObj, plan, approved, skipPreActions, skipPostActions,
			ignoreErrors, dryRun)
	}

	var finishedCh <-chan bool

	switch action {
	case constants.DagActionTemplate, constants.DagActionClean, constants.DagActionOutput,
		constants.DagActionInstall:
		finishedCh = d.walkDown(processCh, doneCh)
	case constants.DagActionDelete:
		// first walk down the DAG to load outputs and build local registries for the kapps, then walk
		// up it executing the marked ones
		_, err := printer.Fprintln("[yellow]Loading kapp outputs...")
		if err != nil {
			return errors.WithStack(err)
		}

		err = initLocalRegistries(d, numWorkers, stackObj, action, approved, dryRun)
		if err != nil {
			return errors.WithStack(err)
		}

		_, err = printer.Fprintf("[green]Kapp outputs loaded\n\n")
		if err != nil {
			return errors.WithStack(err)
		}

		finishedCh = d.walkUp(processCh, doneCh)
	default:
		return fmt.Errorf("Invalid action on DAG: %s", action)
	}

	log.Logger.Debug("Blocking waiting for the DAG to finish processing...")

	for {
		// Note: Do NOT add a case for doneCh or it'll introduce a race that prevents the DAG from
		// updating the status of each node
		select {
		case err := <-errCh:
			return errors.Wrapf(err, "Error processing kapp")
		case <-finishedCh:
			log.Logger.Infof("Finished processing kapps")
			return nil
		}
	}
}

// Traverses the DAG printing vars for all marked nodes, optionally suppressing output for certain keys
func (d *Dag) ExecuteGetVars(action string, stackObj interfaces.IStack, loadOutputs bool, suppress []string) error {
	numWorkers := config.CurrentConfig.NumWorkers

	processCh := make(chan NamedNode, numWorkers)
	doneCh := make(chan NamedNode)
	errCh := make(chan error)

	log.Logger.Infof("Executing DAG with action=%s", action)

	// create the worker pool
	for w := int(0); w < numWorkers; w++ {
		go varsWorker(processCh, doneCh, errCh, stackObj, suppress)
	}

	var finishedCh <-chan bool

	if loadOutputs {
		// initialise local registries to make outputs available
		err := initLocalRegistries(d, numWorkers, stackObj, action, false, false)
		if err != nil {
			return errors.WithStack(err)
		}
	} else {
		log.Logger.Debugf("Skipping loading outputs")
	}

	switch action {
	case constants.DagActionVars:
		finishedCh = d.walkDown(processCh, doneCh)
	default:
		return fmt.Errorf("Invalid action on DAG: %s", action)
	}

	log.Logger.Debug("Blocking waiting for the DAG to finish processing...")

	for {
		// Note: Do NOT add a case for doneCh or it'll introduce a race that prevents the DAG from
		// updating the status of each node
		select {
		case err := <-errCh:
			return errors.Wrapf(err, "Error processing kapp")
		case <-finishedCh:
			log.Logger.Infof("Finished processing kapps")
			return nil
		}
	}
}

// Creates a pool of workers to populate the local registries on installables in the DAG
func initLocalRegistries(dagObj *Dag, numWorkers int, stackObj interfaces.IStack, action string,
	approved bool, dryRun bool) error {

	log.Logger.Debug("Walking down the DAG to initialise local registries")

	// create a new set of channels for the workers
	processCh := make(chan NamedNode, numWorkers)
	doneCh := make(chan NamedNode)
	errCh := make(chan error)

	for w := int(0); w < numWorkers; w++ {
		go registryWorker(dagObj, processCh, doneCh, errCh, stackObj, action, approved, dryRun)
	}

	finishedCh := dagObj.walkDown(processCh, doneCh)

	for {
		// Note: Do NOT add a case for doneCh or it'll introduce a race that prevents the DAG from
		// updating the status of each node
		select {
		case err := <-errCh:
			return errors.Wrapf(err, "Error processing registry workers")
		case <-finishedCh:
			log.Logger.Infof("Finished processing registry workers")
			return nil
		}
	}
}

func registryWorker(dagObj *Dag, processCh <-chan NamedNode, doneCh chan<- NamedNode, errCh chan error,
	stackObj interfaces.IStack, action string, approved bool, dryRun bool) {

	for node := range processCh {
		installableObj := node.installableObj

		addParentRegistries(dagObj, node, errCh)

		kappRootDir := installableObj.GetCacheDir()
		log.Logger.Infof("Registry worker received kapp '%s' in %s for processing", installableObj.FullyQualifiedId(), kappRootDir)

		_, err := os.Stat(kappRootDir)
		if err != nil {
			msg := fmt.Sprintf("Kapp '%s' doesn't exist in the cache at '%s'", installableObj.Id(), kappRootDir)
			log.Logger.Warn(msg)
			errCh <- errors.Wrap(err, msg)
			return
		}

		// kapp exists, Instantiate an installer in case we need it (for now, this will always be a RunUnit installer)
		installerImpl, err := installer.New(installer.RunUnit, stackObj.GetProvider())
		if err != nil {
			errCh <- errors.Wrapf(err, "Error instantiating installer for "+
				"kapp '%s'", installableObj.Id())
			return
		}

		// template the kapp's descriptor, including the global registry
		templatedVars, err := stackObj.GetTemplatedVars(installableObj, map[string]interface{}{})
		if err != nil {
			errCh <- errors.WithStack(err)
			return
		}

		err = installableObj.TemplateDescriptor(templatedVars)
		if err != nil {
			errCh <- errors.WithStack(err)
			return
		}

		// try loading outputs, but don't fail if we can't
		outputs, err := getOutputs(installableObj, stackObj, installerImpl, true, dryRun)
		if err != nil {
			errCh <- errors.WithStack(err)
			return
		}

		// add outputs to the kapp
		err = addOutputsToRegistry(installableObj, outputs, installableObj.GetLocalRegistry(), true)
		if err != nil {
			errCh <- errors.WithStack(err)
			return
		}

		// and also to the stack's registry (but only with fully-qualified keys)
		err = addOutputsToRegistry(installableObj, outputs, stackObj.GetRegistry(), false)
		if err != nil {
			errCh <- errors.WithStack(err)
			return
		}

		log.Logger.Tracef("Registry worker finished processing kapp '%s' (node=%#v)", installableObj.FullyQualifiedId(),
			node)
		doneCh <- node
		log.Logger.Tracef("Registry worker end of loop for kapp '%s'", installableObj.FullyQualifiedId())
	}
}

// Processes an installable, either installing/deleting it, running post actions or
// loading its outputs, etc.
func worker(dagObj *Dag, processCh <-chan NamedNode, doneCh chan<- NamedNode, errCh chan error,
	action string, stackObj interfaces.IStack, plan bool, approved bool, skipPreActions bool, skipPostActions bool,
	ignoreErrors bool, dryRun bool) {

	for node := range processCh {
		installableObj := node.installableObj

		addParentRegistries(dagObj, node, errCh)

		kappRootDir := installableObj.GetCacheDir()
		log.Logger.Infof("Worker received kapp '%s' in %s for processing", installableObj.FullyQualifiedId(), kappRootDir)

		// todo - print (to stdout) details of the kapp being executed

		_, err := os.Stat(kappRootDir)
		if err != nil {
			msg := fmt.Sprintf("Kapp '%s' doesn't exist in the cache at '%s'", installableObj.Id(), kappRootDir)
			log.Logger.Warn(msg)
			errCh <- errors.Wrap(err, msg)
			return
		}

		// Default to the make installer
		installerName := installer.RunUnit

		log.Logger.Debugf("Instantiating a new '%s' installer for kapp '%s'", installerName, installableObj.Id())

		// kapp exists, Instantiate an installer in case we need it (for now, this will always be a Make installer)
		installerImpl, err := installer.New(installerName, stackObj.GetProvider())
		if err != nil {
			errCh <- errors.Wrapf(err, "Error instantiating installer for "+
				"kapp '%s'", installableObj.Id())
			return
		}

		var runSteps []structs.RunStep

		switch action {
		case constants.DagActionInstall:
			installOrDelete(true, dagObj, node, installerImpl, stackObj, plan, approved, skipPreActions,
				skipPostActions, ignoreErrors, dryRun, errCh)
		case constants.DagActionDelete:
			installOrDelete(false, dagObj, node, installerImpl, stackObj, plan, approved, skipPreActions,
				skipPostActions, ignoreErrors, dryRun, errCh)
		case constants.DagActionClean:
			if node.marked {
				// template the kapp's descriptor, including the global registry
				templatedVars, err := stackObj.GetTemplatedVars(installableObj,
					map[string]interface{}{})
				err = installableObj.TemplateDescriptor(templatedVars)
				if err != nil {
					errCh <- errors.WithStack(err)
					return
				}

				runSteps, err = installerImpl.Clean(installableObj, stackObj, dryRun)
				if err != nil {
					errCh <- errors.Wrapf(err, "Error cleaning kapp '%s'", installableObj.Id())
					return
				}

				err = executeRunSteps(constants.Clean, runSteps, installableObj, stackObj, installerImpl.Clean, dryRun)
				if err != nil {
					errCh <- errors.Wrapf(err, "Error executing run steps for kapp '%s'", installableObj.Id())
					return
				}
			}
		case constants.DagActionOutput:
			if node.marked {
				// template the kapp's descriptor, including the global registry
				templatedVars, err := stackObj.GetTemplatedVars(installableObj,
					map[string]interface{}{})
				err = installableObj.TemplateDescriptor(templatedVars)
				if err != nil {
					errCh <- errors.WithStack(err)
					return
				}

				runSteps, err = installerImpl.Output(installableObj, stackObj, dryRun)
				if err != nil {
					errCh <- errors.Wrapf(err, "Error generating output for kapp '%s'", installableObj.Id())
					return
				}

				err = executeRunSteps(constants.Output, runSteps, installableObj, stackObj, installerImpl.Output, dryRun)
				if err != nil {
					errCh <- errors.Wrapf(err, "Error executing run steps for kapp '%s'", installableObj.Id())
					return
				}
			}
		case constants.DagActionTemplate:
			// Template nodes before trying to get the output in case getting the output relies on templated
			// files, e.g. terraform backends
			if node.marked {
				err = renderKappTemplates(stackObj, installableObj, false, dryRun)
				if err != nil {
					if ignoreErrors {
						log.Logger.Warnf("Ignoring error templating kapp: %#v", err)
						doneCh <- node
					} else {
						errCh <- errors.WithStack(err)
					}
					return
				}
			}

			// template the kapp's descriptor, including the global registry
			templatedVars, err := stackObj.GetTemplatedVars(installableObj,
				map[string]interface{}{})
			err = installableObj.TemplateDescriptor(templatedVars)
			if err != nil {
				errCh <- errors.WithStack(err)
				return
			}

			// try loading outputs, but don't fail if we can't
			outputs, err := getOutputs(installableObj, stackObj, installerImpl, true, dryRun)
			if err != nil {
				if ignoreErrors {
					log.Logger.Warnf("Ignoring error getting outputs: %#v", err)
					doneCh <- node
				} else {
					errCh <- errors.WithStack(err)
				}
				return
			}

			// add outputs to the kapp
			err = addOutputsToRegistry(installableObj, outputs, installableObj.GetLocalRegistry(), true)
			if err != nil {
				errCh <- errors.WithStack(err)
				return
			}

			// and also to the stack's registry (but only with fully-qualified keys)
			err = addOutputsToRegistry(installableObj, outputs, stackObj.GetRegistry(), false)
			if err != nil {
				errCh <- errors.WithStack(err)
				return
			}

			// only template marked nodes
			if node.marked {
				err = renderKappTemplates(stackObj, installableObj, true, dryRun)
				if err != nil {
					if ignoreErrors {
						log.Logger.Warnf("Ignoring error templating kapp: %#v", err)
						doneCh <- node
					} else {
						errCh <- errors.WithStack(err)
					}
					return
				}
			}
		}

		log.Logger.Tracef("Worker finished processing kapp '%s' (node=%#v)", installableObj.FullyQualifiedId(),
			node)
		doneCh <- node
		log.Logger.Tracef("Worker end of loop for kapp '%s'", installableObj.FullyQualifiedId())
	}
}

// Prints out the variables for each marked node
func varsWorker(processCh <-chan NamedNode, doneCh chan<- NamedNode, errCh chan error, stackObj interfaces.IStack,
	suppress []string) {

	for node := range processCh {
		installableObj := node.installableObj

		if !node.marked {
			log.Logger.Debugf("Not printing variables for unmarked node: '%s'", installableObj.FullyQualifiedId())
			doneCh <- node
			continue
		}

		kappRootDir := installableObj.GetCacheDir()
		log.Logger.Infof("Worker received kapp '%s' in %s for processing", installableObj.FullyQualifiedId(), kappRootDir)

		// todo - print (to stdout) details of the kapp being executed

		_, err := os.Stat(kappRootDir)
		if err != nil {
			msg := fmt.Sprintf("Kapp '%s' doesn't exist in the cache at '%s'", installableObj.Id(), kappRootDir)
			log.Logger.Warn(msg)
			errCh <- errors.Wrap(err, msg)
			return
		}

		log.Logger.Debugf("Getting variables for kapp '%s'", installableObj.FullyQualifiedId())

		// template the kapp's descriptor, including the global registry
		templatedVars, err := stackObj.GetTemplatedVars(installableObj, map[string]interface{}{})

		if len(suppress) > 0 {
			for _, exclusion := range suppress {
				// trim any leading zeroes for compatibility with how variables are referred to in templates
				exclusion = strings.TrimPrefix(exclusion, ".")
				blanked := utils.BlankNestedMap(map[string]interface{}{}, strings.Split(exclusion, "."))
				log.Logger.Debugf("blanked=%#v", blanked)

				err = mergo.Merge(&templatedVars, blanked, mergo.WithOverride)
				if err != nil {
					errCh <- errors.WithStack(err)
					return
				}
			}
		}

		yamlData, err := yaml.Marshal(&templatedVars)
		if err != nil {
			errCh <- errors.WithStack(err)
			return
		}

		_, err = printer.Fprintf("\n[yellow]***** Start variables for kapp '[bold]%s[reset][yellow]' *****[reset]\n"+
			"%s[yellow]***** End variables for kapp '[bold]%s[reset][yellow]' *****[reset]\n",
			installableObj.FullyQualifiedId(), yamlData, installableObj.FullyQualifiedId())
		if err != nil {
			errCh <- errors.WithStack(err)
			return
		}

		err = installableObj.TemplateDescriptor(templatedVars)
		if err != nil {
			errCh <- errors.WithStack(err)
			return
		}

		kappConfig, err := yaml.Marshal(installableObj.GetDescriptor())
		if err != nil {
			errCh <- errors.WithStack(err)
			return
		}

		_, err = printer.Fprintf("\n[yellow]***** Start config for kapp '[bold]%s[reset][yellow]' *****[reset]\n"+
			"%s[yellow]***** End config for kapp '[bold]%s[reset][yellow]' *****[reset]\n",
			installableObj.FullyQualifiedId(), kappConfig, installableObj.FullyQualifiedId())

		log.Logger.Tracef("Vars worker finished processing kapp '%s' (node=%#v)", installableObj.FullyQualifiedId(),
			node)
		doneCh <- node
		log.Logger.Tracef("Vars worker end of loop for kapp '%s'", installableObj.FullyQualifiedId())
	}
}

// Implements the install action. Nodes that should be processed are installed. All nodes load any outputs
// and merge them with their parents' outputs.
func installOrDelete(install bool, dagObj *Dag, node NamedNode, installerImpl interfaces.IInstaller,
	stackObj interfaces.IStack, plan bool, approved bool, skipPreActions bool, skipPostActions bool, ignoreErrors bool,
	dryRun bool, errCh chan error) {

	installableObj := node.installableObj

	var runSteps []structs.RunStep
	var installerMethod func(installableObj interfaces.IInstallable, stack interfaces.IStack, dryRun bool) ([]structs.RunStep, error)
	var preActions []structs.Action
	var postActions []structs.Action

	actionName := constants.DagActionInstall
	if !install {
		actionName = constants.DagActionDelete
	}

	// render templates in case any are used as outputs for some reason
	err := renderKappTemplates(stackObj, installableObj, true, dryRun)
	if err != nil {
		errCh <- errors.WithStack(err)
		return
	}

	// only plan or process kapps that have been flagged for processing
	if node.marked {
		if plan {
			var unitName string
			if install {
				installerMethod = installerImpl.PlanInstall
				unitName = constants.PlanInstall
			} else {
				installerMethod = installerImpl.PlanDelete
				unitName = constants.PlanDelete
			}

			runSteps, err = installerMethod(installableObj, stackObj, dryRun)
			if err != nil {
				if ignoreErrors {
					log.Logger.Warnf("Ignoring error planning kapp '%s': %#v",
						installableObj.FullyQualifiedId(), err)

					_, err = printer.Fprintf("[yellow]Ignoring error planning '[bold][white]%s[reset][yellow]'.\n",
						installableObj.FullyQualifiedId())
					if err != nil {
						errCh <- errors.WithStack(err)
						return
					}
				} else {
					errCh <- errors.Wrapf(err, "Error planning kapp '%s'", installableObj.Id())
					return
				}
			}

			err = executeRunSteps(unitName, runSteps, installableObj, stackObj, installerMethod, dryRun)
			if err != nil {
				if ignoreErrors {
					log.Logger.Warnf("Ignoring error planning kapp '%s': %#v",
						installableObj.FullyQualifiedId(), err)

					_, err = printer.Fprintf("[yellow]Ignoring error planning '[bold][white]%s[reset][yellow]'.\n",
						installableObj.FullyQualifiedId())
					if err != nil {
						errCh <- errors.WithStack(err)
						return
					}
				} else {
					errCh <- errors.Wrapf(err, "Error planning kapp '%s'", installableObj.Id())
					return
				}
			}
		}

		skipInstallerMethod := false

		// only execute pre actions if approved==true
		if approved {
			if install {
				preActions = installableObj.PreInstallActions()
			} else {
				preActions = installableObj.PreDeleteActions()
			}

			if skipPreActions {
				if len(preActions) > 0 {
					_, err = printer.Fprintf("[yellow]Not executing %d pre actions for '[bold][white]%s[reset][yellow]'. Pass "+
						"`[bold]--%s[reset][yellow]` to execute them\n", len(preActions), installableObj.FullyQualifiedId(), constants.RunPreActions)
					if err != nil {
						errCh <- errors.WithStack(err)
						return
					}
				}
			} else {
				log.Logger.Infof("Will run %d pre %s actions", len(preActions), actionName)

				for _, action := range preActions {
					switch action.Id {
					case constants.ActionSkip:
						log.Logger.Infof("Marking that we should skip running '%s' on installable '%s'",
							actionName, installableObj.FullyQualifiedId())
						skipInstallerMethod = true
					default:
						executeAction(action, installableObj, stackObj, errCh, ignoreErrors, dryRun)
					}
				}
			}
		}

		if approved && !skipInstallerMethod {
			var unitName string
			if install {
				installerMethod = installerImpl.ApplyInstall
				unitName = constants.ApplyInstall
			} else {
				installerMethod = installerImpl.ApplyDelete
				unitName = constants.ApplyDelete
			}

			runSteps, err = installerMethod(installableObj, stackObj, dryRun)
			if err != nil {
				if ignoreErrors {
					log.Logger.Warnf("Ignoring error applying kapp '%s': %#v",
						installableObj.FullyQualifiedId(), err)

					_, err = printer.Fprintf("[yellow]Ignoring error applying '[bold][white]%s[reset][yellow]'.\n",
						installableObj.FullyQualifiedId())
					if err != nil {
						errCh <- errors.WithStack(err)
						return
					}
				} else {
					errCh <- errors.Wrapf(err, "Error processing kapp '%s'", installableObj.Id())
					return
				}
			}

			err = executeRunSteps(unitName, runSteps, installableObj, stackObj, installerMethod, dryRun)
			if err != nil {
				if ignoreErrors {
					log.Logger.Warnf("Ignoring error applying kapp '%s': %#v",
						installableObj.FullyQualifiedId(), err)

					_, err = printer.Fprintf("[yellow]Ignoring error applying '[bold][white]%s[reset][yellow]'.\n",
						installableObj.FullyQualifiedId())
					if err != nil {
						errCh <- errors.WithStack(err)
						return
					}
				} else {
					errCh <- errors.Wrapf(err, "Error processing kapp '%s'", installableObj.Id())
					return
				}
			}
		}
	}

	// get outputs if we've installed the kapp (we assume we even need to get the outputs for skipped kapps since
	// the assumption is that those kapps will have been previously installed and already exist - otherwise they'd
	// be marked as absent not to be installed at all)
	var outputs map[string]interface{}
	if install && approved {
		// fail if outputs don't exist
		outputs, err = getOutputs(installableObj, stackObj, installerImpl, false, dryRun)
		if err != nil {
			errCh <- errors.WithStack(err)
			return
		}
	}

	// build the kapp's local registry
	err = addOutputsToRegistry(installableObj, outputs, installableObj.GetLocalRegistry(), true)
	if err != nil {
		errCh <- errors.WithStack(err)
		return
	}

	// and also to the stack's registry (but only with fully-qualified keys)
	err = addOutputsToRegistry(installableObj, outputs, stackObj.GetRegistry(), false)
	if err != nil {
		errCh <- errors.WithStack(err)
		return
	}

	// rerender templates so they can use kapp outputs (e.g. before adding the paths to rendered templates as provider vars)
	err = renderKappTemplates(stackObj, installableObj, false, dryRun)
	if err != nil {
		errCh <- errors.WithStack(err)
		return
	}

	// only execute post actions if approved==true
	if node.marked && approved {
		if install {
			postActions = installableObj.PostInstallActions()
		} else {
			postActions = installableObj.PostDeleteActions()
		}

		if skipPostActions {
			if len(postActions) > 0 {
				_, err = printer.Fprintf("[yellow]Not executing %d post actions for '[bold][white]%s[reset][yellow]'. Pass "+
					"`[bold]--%s[reset][yellow]` to execute them\n", len(postActions), installableObj.FullyQualifiedId(), constants.RunPostActions)
				if err != nil {
					errCh <- errors.WithStack(err)
					return
				}
			}
		} else {
			log.Logger.Infof("Will run %d post %s actions", len(postActions), actionName)

			for _, action := range postActions {
				executeAction(action, installableObj, stackObj, errCh, ignoreErrors, dryRun)
			}
		}
	}
}

// Executes a list of run steps
func executeRunSteps(unitName string, runSteps []structs.RunStep, installableObj interfaces.IInstallable,
	stackObj interfaces.IStack,
	installerMethod func(installableObj interfaces.IInstallable, stack interfaces.IStack, dryRun bool) ([]structs.RunStep, error),
	dryRun bool) error {

	dryRunPrefix := ""
	if dryRun {
		dryRunPrefix = "[Dry run] "
	}

	_, err := printer.Fprintf("%s[white][bold]%s[reset] - Executing '[white]%s[default]' run steps...\n",
		dryRunPrefix, installableObj.FullyQualifiedId(), unitName)
	if err != nil {
		return errors.WithStack(err)
	}

	var outPath string

	var step structs.RunStep
	// iterate using a counter because we may need to retemplate the run steps during iteration
	for i := 0; i < len(runSteps); i++ {
		step = runSteps[i]

		// evaluate any conditions
		allOk, err := utils.All(step.Conditions)
		if err != nil {
			return errors.WithStack(err)
		}

		if !allOk {
			if config.CurrentConfig.Verbose {
				_, err := printer.Fprintf("Some conditions for the '[white]%s[default]' run step '[white]%s[default]' "+
					"evaluated to false. Won't execute it...\n", installableObj.FullyQualifiedId(), step.Name)
				if err != nil {
					return errors.WithStack(err)
				}
			}

			log.Logger.Infof("Some conditions for run step '%s' evaluated to false for kapp '%s'. Won't execute "+
				"run units for it.", step.Name, installableObj.FullyQualifiedId())
			continue
		}

		args, err := shellwords.Parse(step.Args)
		if err != nil {
			return errors.WithStack(err)
		}

		log.Logger.Infof("Executing run step '%s' for '%s'", step.Name, installableObj.FullyQualifiedId())

		if config.CurrentConfig.Verbose {
			_, err := printer.Fprintf("* %s[white]%s[reset] - Executing command for run step "+
				"'[white]%s[default]':\n%s %s\n\n",
				dryRunPrefix, installableObj.FullyQualifiedId(), step.Name, step.Command, strings.Join(args, " "))
			if err != nil {
				return errors.WithStack(err)
			}
		} else {
			_, err := printer.Fprintf("* %s[white]%s[reset] - Executing run step "+
				"'[white]%s[default]'...\n",
				dryRunPrefix, installableObj.FullyQualifiedId(), step.Name)
			if err != nil {
				return errors.WithStack(err)
			}
		}

		var stdoutBuf, stderrBuf bytes.Buffer
		cmdErr := utils.ExecCommand(step.Command, args, step.EnvVars, &stdoutBuf,
			&stderrBuf, step.WorkingDir, step.Timeout, step.ExpectedExitCode, dryRun)

		log.Logger.Infof("Stdout: %s", stdoutBuf.String())
		log.Logger.Infof("Stderr: %s", stderrBuf.String())

		if step.Print != "" {

			shouldPrint := true
			if step.Print == constants.Verbose && !config.CurrentConfig.Verbose {
				shouldPrint = false
			}

			if shouldPrint {
				stdout := stdoutBuf.String()
				if stdout != "" {
					stdout = fmt.Sprintf("\n\n%s", stdout)
				}

				stderr := stderrBuf.String()
				if stderr != "" {
					stderr = fmt.Sprintf("\n\n%s", stderr)
				}

				if len(stdout) > 0 {
					_, err := printer.Fprintf("\n[yellow][bold]Stdout[reset][yellow] from[reset] '[white]%s[reset]' for '[white]%s[reset]': %s\n",
						step.Name, installableObj.FullyQualifiedId(), stdout)
					if err != nil {
						return errors.WithStack(err)
					}
				} else {
					_, err := printer.Fprintf("\n[yellow]No [bold]stdout[reset][yellow] was written by[reset] '[white]%s[reset]' for '[white]%s[reset]'",
						step.Name, installableObj.FullyQualifiedId())
					if err != nil {
						return errors.WithStack(err)
					}
				}

				if len(stderr) > 0 {
					_, err = printer.Fprintf("\n[yellow][bold]Stderr[reset][yellow] from[reset] '[white]%s[reset]' for '[white]%s[reset]': %s\n",
						step.Name, installableObj.FullyQualifiedId(), stderr)
					if err != nil {
						return errors.WithStack(err)
					}
				} else {
					_, err := printer.Fprintf("\n[yellow]No [bold]stderr[reset][yellow] was written by[reset] '[white]%s[reset]' for '[white]%s[reset]'\n\n",
						step.Name, installableObj.FullyQualifiedId())
					if err != nil {
						return errors.WithStack(err)
					}
				}
			}
		}

		var err2 error

		if step.Stdout != "" {
			outPath = utils.MakePathAbsoluteIfNot(step.Stdout, step.WorkingDir)
			log.Logger.Debugf("Writing stdout from step '%s' to '%s'", step.Name, outPath)
			err2 = ioutil.WriteFile(outPath, stdoutBuf.Bytes(), 0644)
		}

		if step.Stderr != "" {
			outPath = utils.MakePathAbsoluteIfNot(step.Stderr, step.WorkingDir)
			log.Logger.Debugf("Writing stderr from step '%s' to '%s'", step.Name, outPath)
			err2 = ioutil.WriteFile(outPath, stderrBuf.Bytes(), 0644)
		}

		// the original error is more important to return, so return that. We should write files
		// before returning it though
		if cmdErr != nil {
			if step.IgnoreErrors {
				log.Logger.Infof("Ignoring error running step '%s' because ignore_errors is true: %v",
					step.Name, cmdErr)

				_, err := printer.Fprintf("* %s[white]%s[reset] - Ignoring errors running step '[white]%s[default]'...\n",
					dryRunPrefix, installableObj.FullyQualifiedId(), step.Name)
				if err != nil {
					return errors.WithStack(err)
				}
			} else {
				return errors.WithStack(cmdErr)
			}
		}

		if err2 != nil {
			return errors.WithStack(err2)
		}

		if step.LoadOutputs && installableObj.HasOutputs() {
			log.Logger.Debugf("Loading outputs for step '%s'", step.Name)
			// load any outputs we can, parse them and add values to the registry
			outputs, err := installableObj.GetOutputs(true, dryRun)
			if err != nil {
				return errors.Wrapf(err, "Error loading the output of kapp '%s'", installableObj.Id())
			}

			// add outputs to the kapp's registry
			err = addOutputsToRegistry(installableObj, outputs, installableObj.GetLocalRegistry(), true)
			if err != nil {
				return errors.WithStack(err)
			}

			// and also to the stack's registry (but only with fully-qualified keys)
			err = addOutputsToRegistry(installableObj, outputs, stackObj.GetRegistry(), false)
			if err != nil {
				return errors.WithStack(err)
			}

			// rerender templates (which will also remerge the kapp's config) in case subsequent steps access outputs
			// we've just loaded in templates they use
			err = renderKappTemplates(stackObj, installableObj, false, dryRun)
			if err != nil {
				return errors.WithStack(err)
			}

			// rerender the run steps
			runSteps, err = installerMethod(installableObj, stackObj, dryRun)
			if err != nil {
				return errors.WithStack(err)
			}
		}
	}

	return nil
}

// Makes a kapp generate its output then loads and returns them
func getOutputs(installableObj interfaces.IInstallable, stackObj interfaces.IStack,
	installerImpl interfaces.IInstaller, ignoreMissing bool, dryRun bool) (map[string]interface{}, error) {
	var outputs map[string]interface{}

	// try to load kapp outputs and fail if we can't (assume we only need to do this when installing)
	if installableObj.HasOutputs() {
		// run the output target to write outputs to files
		runSteps, err := installerImpl.Output(installableObj, stackObj, dryRun)
		if err != nil {
			return nil, errors.Wrapf(err, "Error writing output for kapp '%s'", installableObj.Id())
		}

		err = executeRunSteps(constants.Output, runSteps, installableObj, stackObj, installerImpl.Output, dryRun)
		if err != nil {
			return nil, errors.Wrapf(err, "Error executing run steps for kapp '%s'", installableObj.Id())
		}

		// load and parse outputs
		outputs, err = installableObj.GetOutputs(ignoreMissing, dryRun)
		if err != nil {
			return nil, errors.Wrapf(err, "Error loading the output of kapp '%s'", installableObj.Id())
		}
	}

	return outputs, nil
}

// Instantiate a new local registry and add values from the parent registries to it. If the
// parent's manifest ID is different to the current node's manifest ID registry keys for
// non fully-qualified installable IDs will be deleted from the registry before merging. In
// all cases the special value 'this' will not be merged either.
func addParentRegistries(dagObj *Dag, node NamedNode, errCh chan<- error) {
	localRegistry := registry.New()

	// clear any default values from the registry before using it
	for k := range localRegistry.AsMap() {
		localRegistry.Delete(k)
	}

	parents := dagObj.graph.To(node.ID())

	for parents.Next() {
		parent := parents.Node().(NamedNode)

		parentRegistry := parent.installableObj.GetLocalRegistry()

		// if may not be set, e.g. if we ignored errors while creating the workspace
		if parentRegistry == nil {
			continue
		}

		for k, v := range parentRegistry.AsMap() {
			err := localRegistry.Set(k, v)
			if err != nil {
				errCh <- errors.WithStack(err)
				return
			}
		}

		// if the parent was in a different manifest, strip out all non fully-qualified registry
		// entries
		if parent.installableObj.ManifestId() != node.installableObj.ManifestId() {
			deleteNonFullyQualifiedOutputs(localRegistry)
		}

		// always delete the special key 'this'
		deleteSpecialThisOutput(localRegistry)
	}

	node.installableObj.SetLocalRegistry(localRegistry)
}

// Executes pre/post kapp actions
func executeAction(action structs.Action, installableObj interfaces.IInstallable,
	stackObj interfaces.IStack, errCh chan error, ignoreErrors bool, dryRun bool) {
	log.Logger.Infof("Executing action '%s' for installable '%s'", action, installableObj.FullyQualifiedId())

	_, err := printer.Fprintf("[white][bold]%s[reset] - Executing the '[white]%s[reset]' action...\n",
		installableObj.FullyQualifiedId(), action.Id)
	if err != nil {
		errCh <- errors.WithStack(err)
		return
	}

	switch action.Id {
	case constants.ActionClusterUpdate:
		err := cluster.UpdateCluster(stackObj, true, dryRun)
		if err != nil {
			if ignoreErrors {
				log.Logger.Warnf("Ignoring error executing '%s' action for kapp '%s'", action.Id, installableObj.FullyQualifiedId())

				_, err := printer.Fprintf("* [white]%s[reset] - [yellow]Ignoring error executing the "+
					"'[white]%s[reset][yellow]' action.\n", installableObj.FullyQualifiedId(), action.Id)
				if err != nil {
					errCh <- err
					return
				}
			} else {
				log.Logger.Warnf("Error executing '%s' action for kapp '%s'", action.Id, installableObj.FullyQualifiedId())
				errCh <- errors.Wrapf(err, "Error updating cluster, triggered by kapp '%s'",
					installableObj.Id())
				return
			}
		}
	case constants.ActionClusterDelete:
		err := stackObj.GetProvisioner().Delete(true, dryRun)
		if err != nil {
			if ignoreErrors {
				log.Logger.Warnf("Ignoring error executing '%s' action for kapp '%s'", action.Id, installableObj.FullyQualifiedId())
				_, err := printer.Fprintf("* [white]%s[reset] - [yellow]Ignoring error executing the "+
					"'[white]%s[reset][yellow]' action.\n", installableObj.FullyQualifiedId(), action.Id)
				if err != nil {
					errCh <- err
					return
				}
			} else {
				log.Logger.Warnf("Error executing '%s' action for kapp '%s'", action.Id, installableObj.FullyQualifiedId())
				errCh <- errors.Wrapf(err, "Error deleting cluster, triggered by kapp '%s'",
					installableObj.Id())
				return
			}
		}
	case constants.ActionAddProviderVarsFiles:
		// todo - run each path through the templater
		for _, path := range action.Params {
			if !filepath.IsAbs(path) {
				// convert the relative path to absolute
				path = filepath.Join(installableObj.GetConfigFileDir(), path)
			}

			log.Logger.Debugf("Adding provider vars dir: %s", path)
			stackObj.GetProvider().AddVarsPath(path)
		}

		// refresh the provider vars so the extra vars files we've just added are loaded
		err := stackObj.RefreshProviderVars()
		if err != nil {
			if ignoreErrors {
				log.Logger.Warnf("Ignoring error executing '%s' action for kapp '%s'", action.Id, installableObj.FullyQualifiedId())
				_, err := printer.Fprintf("* [white]%s[reset] - [yellow]Ignoring error executing the "+
					"'[white]%s[reset][yellow]' action.\n", installableObj.FullyQualifiedId(), action.Id)
				if err != nil {
					errCh <- err
					return
				}
			} else {
				log.Logger.Warnf("Error executing '%s' action for kapp '%s'", action.Id, installableObj.FullyQualifiedId())
				errCh <- errors.WithStack(err)
				return
			}
		}
	}
}

// Deletes all outputs from the registry that aren't fully qualified
func deleteNonFullyQualifiedOutputs(registry interfaces.IRegistry) {
	outputs, ok := registry.Get(constants.RegistryKeyOutputs)
	if !ok {
		return
	}

	// iterate through all the keys for those that aren't fully qualified and delete them
	for k := range outputs.(map[string]interface{}) {
		if !strings.Contains(k, constants.TemplateNamespaceSeparator) {
			fullKey := strings.Join([]string{
				constants.RegistryKeyOutputs, k}, constants.RegistryFieldSeparator)
			registry.Delete(fullKey)
		}
	}
}

// deletes the special constant key "this" from the registry
func deleteSpecialThisOutput(registry interfaces.IRegistry) {
	registry.Delete(strings.Join([]string{constants.RegistryKeyOutputs,
		constants.RegistryKeyThis}, constants.RegistryFieldSeparator))
}

// Adds output from an installable to the registry
// If `includeLocalKeys` is true, additional keys will be added to the registry for access from within a kapp (e.g.
// `.outputs.this`, etc.
func addOutputsToRegistry(installableObj interfaces.IInstallable, outputs map[string]interface{},
	registry interfaces.IRegistry, includeLocalKeys bool) error {

	// We convert kapp IDs to have underscores because Go's templating library throws its toys out
	// the pram when it find a map key with a hyphen in. K8s is the opposite, so this seems like
	// the least worst way of accommodating both
	underscoredInstallableId := strings.Replace(installableObj.Id(), "-", "_", -1)
	underscoredInstallableFQId := strings.Replace(installableObj.FullyQualifiedId(), "-", "_", -1)
	underscoredInstallableFQId = strings.Replace(underscoredInstallableFQId, constants.NamespaceSeparator,
		constants.TemplateNamespaceSeparator, -1)

	prefixes := make([]string, 0)

	if includeLocalKeys {
		// "outputs.this"
		prefixes = append(prefixes, strings.Join([]string{constants.RegistryKeyOutputs, constants.RegistryKeyThis},
			constants.RegistryFieldSeparator))

		// short prefix - can be used by other kapps in the manifest
		prefixes = append(prefixes, strings.Join([]string{constants.RegistryKeyOutputs, underscoredInstallableId},
			constants.RegistryFieldSeparator))
	}

	// always add keys for the fully-qualified kapp ID
	prefixes = append(prefixes, strings.Join([]string{constants.RegistryKeyOutputs, underscoredInstallableFQId},
		constants.RegistryFieldSeparator))

	// store the output under various keys
	for outputId, output := range outputs {
		for _, prefix := range prefixes {
			underscoredOutputId := strings.Replace(outputId, "-", "_", -1)
			key := strings.Join([]string{prefix, underscoredOutputId}, constants.RegistryFieldSeparator)
			err := registry.Set(key, output)
			if err != nil {
				return errors.WithStack(err)
			}
		}
	}

	return nil
}

// Renders templates for a kapp
func renderKappTemplates(stackObj interfaces.IStack, installableObj interfaces.IInstallable,
	printMessage bool, dryRun bool) error {

	// merge all the vars required to render the kapp's sugarkube.yaml file
	templatedVars, err := stackObj.GetTemplatedVars(installableObj, map[string]interface{}{})
	if err != nil {
		return errors.WithStack(err)
	}

	// template the descriptor again in case variables refer to outputs
	err = installableObj.TemplateDescriptor(templatedVars)
	if err != nil {
		return errors.WithStack(err)
	}

	// get the updated template vars
	templatedVars, err = stackObj.GetTemplatedVars(installableObj, map[string]interface{}{})
	if err != nil {
		return errors.WithStack(err)
	}

	err = installableObj.RenderTemplates(templatedVars, stackObj.GetConfig(),
		dryRun)
	if err != nil {
		return errors.WithStack(err)
	}

	log.Logger.Tracef("Vars after merging template paths: %#v", templatedVars)

	// remerge and template the kapp's descriptor so it can access the paths of any rendered templates
	err = installableObj.TemplateDescriptor(templatedVars)
	if err != nil {
		return errors.WithStack(err)
	}

	log.Logger.Tracef("Template descriptor after merging template paths: %#v", installableObj.GetDescriptor())

	templatedPaths := make([]string, 0)
	for _, template := range installableObj.GetDescriptor().Templates {
		templatedPaths = append(templatedPaths, template.RenderedPath)
	}

	if config.CurrentConfig.Verbose && printMessage {
		_, err := printer.Fprintf("  Rendered templates for '[white][bold]%s[reset]' to '%s'\n",
			installableObj.FullyQualifiedId(), strings.Join(templatedPaths, "', '"))
		if err != nil {
			return errors.WithStack(err)
		}
	}

	return nil
}
