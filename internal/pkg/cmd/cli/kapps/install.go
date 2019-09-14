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

package kapps

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/sugarkube/sugarkube/internal/pkg/constants"
	"github.com/sugarkube/sugarkube/internal/pkg/log"
	"github.com/sugarkube/sugarkube/internal/pkg/plan"
	"github.com/sugarkube/sugarkube/internal/pkg/printer"
	"github.com/sugarkube/sugarkube/internal/pkg/program"
	"github.com/sugarkube/sugarkube/internal/pkg/provisioner"
	"github.com/sugarkube/sugarkube/internal/pkg/stack"
	"github.com/sugarkube/sugarkube/internal/pkg/structs"
)

type installCmd struct {
	workspaceDir string
	dryRun       bool
	approved     bool
	oneShot      bool
	//force               bool
	skipTemplating      bool
	runActions          bool
	skipActions         bool
	runPreActions       bool
	runPostActions      bool
	establishConnection bool
	includeParents      bool
	noValidate          bool
	stackName           string
	stackFile           string
	provider            string
	provisioner         string
	profile             string
	account             string
	cluster             string
	region              string
	includeSelector     []string
	excludeSelector     []string
	onlineTimeout       uint32
	readyTimeout        uint32
}

func newInstallCmd() *cobra.Command {
	c := &installCmd{}

	cmd := &cobra.Command{
		Use:   "install [flags] [stack-file] [stack-name] [workspace-dir]",
		Short: fmt.Sprintf("Install kapps into a cluster"),
		Long: `Install kapps in a target cluster according to manifests.

Kapps are installed using two phases - the first expects kapps to plan their
changes but not actually perform any. For example, during this phase any
kapps that use Terraform will run 'terraform plan'. This is the default action for 
this command. If you're running Sugarkube in a CI/CD system and want to inspect 
the plan before applying it you could grep stdout after running this pass.

To actually install/apply kapps, rerun the command passing '--yes'.

For convenience if you invoke Sugarkube passing '--one-shot' it will run both 
phases sequentially.

Dry run mode differs to the planning phase by not actually running kapps at all - 
Sugarkube will just print out how it would invoke each kapp. Dry run mode is 
designed to be fast.

For Kubernetes clusters with a non-public API server, the provisioner may need 
to set up connectivity to make it accessible to Sugarkube (e.g. by setting up 
SSH port forwarding via a bastion). This happens automatically when a cluster 
is created or updated by Sugarkube, but if you're installing individual kapps 
you may need to pass the '--connect' flag to make Sugarkube go through that
process before installing the selected kapps.
`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) < 3 {
				return errors.New("some required arguments are missing")
			} else if len(args) > 3 {
				return errors.New("too many arguments supplied")
			}
			c.stackFile = args[0]
			c.stackName = args[1]
			c.workspaceDir = args[2]

			err1 := c.run()
			// shutdown any SSH port forwarding then return the error
			if stackObj != nil {
				err2 := stackObj.GetProvisioner().Close()
				if err2 != nil {
					return errors.WithStack(err2)
				}
			}

			if err1 != nil {
				if _, silent := errors.Cause(err1).(program.SilentError); !silent {
					_, _ = printer.Fprint("\n[red][bold]Error installing kapp. Aborting.\n")
				}
				return errors.WithStack(err1)
			}

			return nil
		},
	}

	f := cmd.Flags()
	f.BoolVarP(&c.dryRun, "dry-run", "n", false, "show what would happen but don't create a cluster")
	f.BoolVarP(&c.approved, constants.YesFlag, "y", false, "actually install kapps. If false, kapps will be expected to plan "+
		"their changes but not make any destrucive changes (e.g. should run 'terraform plan', etc. but not apply it).")
	f.BoolVar(&c.oneShot, "one-shot", false, "invoke each kapp as if --yes hadn't been given then immediately again as if it had "+
		"to plan and install kapps in a single pass")
	f.BoolVar(&c.includeParents, "parents", false, "process all parents of all selected kapps as well")
	//f.BoolVar(&c.force, "force", false, "don't require a cluster diff, just blindly install/delete all the kapps "+
	//	"defined in a manifest(s)/stack config, even if they're already present/absent in the target cluster")
	f.BoolVarP(&c.skipTemplating, "no-template", "t", false, "skip writing templates for kapps before installing them")
	f.BoolVar(&c.noValidate, "no-validate", false, "don't validate kapps")
	f.BoolVar(&c.runActions, "run-actions", false, "run pre- and post-actions in kapps")
	f.BoolVar(&c.skipActions, "skip-actions", false, "skip pre- and post-actions in kapps")
	f.BoolVar(&c.runPreActions, constants.RunPreActions, false, "run pre actions in kapps")
	f.BoolVar(&c.runPostActions, constants.RunPostActions, false, "run post actions in kapps")
	f.BoolVar(&c.establishConnection, "connect", false, "establish a connection to the API server if it's not publicly accessible")
	f.StringVar(&c.provider, "provider", "", "name of provider, e.g. aws, local, etc.")
	f.StringVar(&c.provisioner, "provisioner", "", "name of provisioner, e.g. kops, minikube, etc.")
	f.StringVar(&c.profile, "profile", "", "launch profile, e.g. dev, test, prod, etc.")
	f.StringVarP(&c.cluster, "cluster", "c", "", "name of cluster to launch, e.g. dev1, dev2, etc.")
	f.StringVarP(&c.account, "account", "a", "", "string identifier for the account to launch in (for providers that support it)")
	f.StringVarP(&c.region, "region", "r", "", "name of region (for providers that support it)")
	f.StringArrayVarP(&c.includeSelector, "include", "i", []string{},
		fmt.Sprintf("only process specified kapps (can specify multiple, formatted 'manifest-id:kapp-id' or 'manifest-id:%s' for all)",
			constants.WildcardCharacter))
	f.StringArrayVarP(&c.excludeSelector, "exclude", "x", []string{},
		fmt.Sprintf("exclude individual kapps (can specify multiple, formatted 'manifest-id:kapp-id' or 'manifest-id:%s' for all)",
			constants.WildcardCharacter))
	f.Uint32Var(&c.onlineTimeout, "online-timeout", 600, "max number of seconds to wait for the cluster to come online")
	f.Uint32Var(&c.readyTimeout, "ready-timeout", 600, "max number of seconds to wait for the cluster to become ready")
	return cmd
}

func (c *installCmd) run() error {

	// CLI overrides - will be merged with any loaded from a stack config file
	cliStackConfig := &structs.StackFile{
		Provider:    c.provider,
		Provisioner: c.provisioner,
		Profile:     c.profile,
		Cluster:     c.cluster,
		Region:      c.region,
		Account:     c.account,
	}

	var err error

	stackObj, err = stack.BuildStack(c.stackName, c.stackFile, cliStackConfig)
	if err != nil {
		return errors.WithStack(err)
	}

	stackObj.GetConfig().SetReadyTimeout(c.readyTimeout)
	stackObj.GetConfig().SetOnlineTimeout(c.onlineTimeout)

	dryRunPrefix := ""
	if c.dryRun {
		dryRunPrefix = "[Dry run] "
	}

	dagObj, err := plan.BuildDagForSelected(stackObj, c.workspaceDir, c.includeSelector, c.excludeSelector, c.includeParents)
	if err != nil {
		return errors.WithStack(err)
	}

	_, err = printer.Fprintln("")
	if err != nil {
		return errors.WithStack(err)
	}

	// if a user selected to run either pre- or post- actions, set the runActions flag if they didn't set it explicitly
	if !c.runActions {
		if c.runPostActions || c.runPreActions {
			c.runActions = true
		}
	}

	err = CatchMistakes(dagObj, c.runActions, c.skipActions, c.noValidate)
	if err != nil {
		return errors.WithStack(err)
	}

	if c.establishConnection {
		err = establishConnection(c.dryRun, dryRunPrefix)
		if err != nil {
			return errors.WithStack(err)
		}
	}

	shouldPlan := false
	approved := false

	if c.oneShot {
		shouldPlan = true
		approved = true

		_, err = printer.Fprintf("[yellow]Running installers for selected kapps in a single pass " +
			"([bold]one shot[reset][yellow])\n")
		if err != nil {
			return errors.WithStack(err)
		}
	} else {
		// frig the messaging to hide the details of how one-shot works
		if c.approved {
			approved = true
			_, err = printer.Fprintf("[yellow]Running installers for selected kapps with [bold]APPROVED=true\n")
			if err != nil {
				return errors.WithStack(err)
			}
		} else {
			shouldPlan = true
			_, err = printer.Fprintf("[yellow]Running installers for selected kapps with [bold]APPROVED=false\n")
			if err != nil {
				return errors.WithStack(err)
			}
		}
	}

	_, err = printer.Fprintf("Enable logging to see the exact parameters passed to each kapp (with `-l info`)\n\n")
	if err != nil {
		return errors.WithStack(err)
	}

	err = dagObj.Execute(constants.DagActionInstall, stackObj, shouldPlan, approved,
		!(c.runPreActions || c.runActions), !(c.runPostActions || c.runActions),
		false, c.dryRun)
	if err != nil {
		return errors.WithStack(err)
	}

	_, err = printer.Fprintf("\n")
	if err != nil {
		return errors.WithStack(err)
	}

	if !approved {
		_, err = printer.Fprintln("[green]All installation plans completed successfully!")
		if err != nil {
			return errors.WithStack(err)
		}
		_, err = printer.Fprintf("[white][bold]Note: [reset]No destructive changes were made. To actually "+
			"install kapps, rerun this command passing [cyan][bold]--%s[reset]\n", constants.YesFlag)
		if err != nil {
			return errors.WithStack(err)
		}
	} else {
		_, err = printer.Fprintf("%s[green]Kapps successfully installed!\n", dryRunPrefix)
		if err != nil {
			return errors.WithStack(err)
		}
	}

	return nil
}

// Establish a connection to the cluster if necessary
func establishConnection(dryRun bool, dryRunPrefix string) error {
	log.Logger.Infof("%sEstablishing connectivity to the API server",
		dryRunPrefix)
	if !dryRun {
		isOnline, err := provisioner.IsAlreadyOnline(stackObj.GetProvisioner(), dryRun)
		if err != nil {
			return errors.WithStack(err)
		}

		if !isOnline {
			log.Logger.Warnf("Cluster '%s' isn't online. Won't try to establish connectivity its "+
				"API server", stackObj.GetConfig().GetCluster())
		}
	}

	return nil
}

// Test whether any kapps have actions and if so explicitly require users to either opt to run or skip them. Also validate
// kapps if users want to
func CatchMistakes(dagObj *plan.Dag, runActions bool, skipActions bool, noValidate bool) error {
	// if no action flags were given, check whether any installables have actions. If they do
	// return an error - the user must explicitly choose whether to run or skip them.
	if !runActions && !skipActions {
		installables := dagObj.GetInstallables()
		for _, installableObj := range installables {
			if installableObj.HasActions() {
				_, err := printer.Fprintf("[red]Kapp '[white]%s[reset][red]' has pre-/post- actions. You must "+
					"explicitly choose whether to run them (with `[bold]--run-actions[reset][red]`, "+
					"`[bold]--run-pre-actions[reset][red]` or `[bold]--run-post-actions[reset][red]`) or skip them "+
					"(with `[bold]--skip-actions[reset][red]`).\n", installableObj.FullyQualifiedId())
				if err != nil {
					return errors.WithStack(err)
				}
				return program.SilentError{}
			}
		}
	}

	if !noValidate {
		err := Validate(dagObj)
		if err != nil {
			return errors.WithStack(err)
		}

		_, err = printer.Fprintln("")
		if err != nil {
			return errors.WithStack(err)
		}
	}

	return nil
}
