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

package kapps

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/sugarkube/sugarkube/internal/pkg/cmd"
	"github.com/sugarkube/sugarkube/internal/pkg/constants"
	"github.com/sugarkube/sugarkube/internal/pkg/plan"
	"github.com/sugarkube/sugarkube/internal/pkg/printer"
	"github.com/sugarkube/sugarkube/internal/pkg/stack"
	"github.com/sugarkube/sugarkube/internal/pkg/structs"
)

type deleteCommand struct {
	workspaceDir        string
	dryRun              bool
	approved            bool
	oneShot             bool
	ignoreErrors        bool
	skipTemplating      bool
	runActions          bool
	noActions           bool
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
}

func newDeleteCommand() *cobra.Command {
	c := &deleteCommand{}

	usage := "delete [flags] [stack-file] [stack-name] [workspace-dir]"

	command := &cobra.Command{
		Use:   usage,
		Short: fmt.Sprintf("Delete kapps from a cluster"),
		Long: `Delete kapps from a target cluster according to manifests.

For Kubernetes clusters with a non-public API server, the provisioner may need 
to set up connectivity to make it accessible to Sugarkube (e.g. by setting up 
SSH port forwarding via a bastion). This happens automatically when a cluster 
is created or updated by Sugarkube, but if you're deleting individual kapps 
you may need to pass the '--connect' flag to make Sugarkube go through that
process before deleting the selected kapps.
`,
		Aliases: []string{"uninstall"},
		RunE: func(command *cobra.Command, args []string) error {
			err := cmd.ValidateNumArgs(args, 3, usage)
			if err != nil {
				return errors.WithStack(err)
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
				return errors.WithStack(err1)
			}

			return nil
		},
	}

	f := command.Flags()
	f.BoolVarP(&c.dryRun, "dry-run", "n", false, "show what would happen but don't create a cluster")
	f.BoolVarP(&c.approved, "yes", "y", false, "actually delete kapps. If false, kapps will be expected to plan "+
		"their changes but not make any destrucive changes (e.g. should run 'terraform plan', etc. but not apply it).")
	f.BoolVar(&c.oneShot, "one-shot", false, "invoke each kapp with 'APPROVED=false' then "+
		"'APPROVED=true' to delete kapps in a single pass")
	f.BoolVar(&c.ignoreErrors, "ignore-errors", false, "ignore errors deleting kapps")
	f.BoolVar(&c.includeParents, "parents", false, "process all parents of all selected kapps as well")
	f.BoolVarP(&c.skipTemplating, "no-template", "t", false, "skip writing templates for kapps before deleting them")
	f.BoolVar(&c.noValidate, "no-validate", false, "don't validate kapps")
	f.BoolVar(&c.noActions, "no-actions", false, "don't run any pre- and post-actions in kapps")
	f.BoolVar(&c.runActions, "run-actions", false, "run pre- and post-actions in kapps")
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
		fmt.Sprintf("only process specified kapps (can specify multiple, formatted manifest-id:kapp-id or 'manifest-id:%s' for all)",
			constants.WildcardCharacter))
	f.StringArrayVarP(&c.excludeSelector, "exclude", "x", []string{},
		fmt.Sprintf("exclude individual kapps (can specify multiple, formatted manifest-id:kapp-id or 'manifest-id:%s' for all)",
			constants.WildcardCharacter))
	return command
}

func (c *deleteCommand) run() error {

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

	err = CatchMistakes(stackObj, dagObj, c.runActions, c.noActions, c.noValidate)
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

		_, err = printer.Fprintf("%s[yellow]Running deletions for selected kapps in a single pass "+
			"([bold]one shot[reset][yellow])\n", dryRunPrefix)
		if err != nil {
			return errors.WithStack(err)
		}
	} else {
		if c.approved {
			approved = true
			_, err = printer.Fprintf("%s[yellow][bold]Applying deletion[reset][yellow] of selected kapps\n", dryRunPrefix)
			if err != nil {
				return errors.WithStack(err)
			}
		} else {
			shouldPlan = true
			_, err = printer.Fprintf("%s[yellow][bold]Planning deletion[reset][yellow] of selected kapps\n", dryRunPrefix)
			if err != nil {
				return errors.WithStack(err)
			}
		}
	}

	_, err = printer.Fprintf("Enable verbose mode (`-v`) to see the commands being executed (or enable logging " +
		"for even more information)\n\n")
	if err != nil {
		return errors.WithStack(err)
	}

	err = dagObj.Execute(constants.DagActionDelete, stackObj, shouldPlan, approved,
		!(c.runPreActions || c.runActions), !(c.runPostActions || c.runActions),
		c.ignoreErrors, c.dryRun)
	if err != nil {
		return errors.WithStack(err)
	}

	_, err = printer.Fprintln("")
	if err != nil {
		return errors.WithStack(err)
	}

	if !approved {
		_, err = printer.Fprintln("[green]All deletion plans completed successfully!")
		if err != nil {
			return errors.WithStack(err)
		}
		_, err = printer.Fprintf("[white][bold]Note: [reset]No destructive changes were made. To actually "+
			"delete kapps, rerun this command passing [cyan][bold]--%s[reset]\n", constants.YesFlag)
		if err != nil {
			return errors.WithStack(err)
		}
	} else {
		_, err = printer.Fprintf("%s[green]Kapps successfully deleted!\n", dryRunPrefix)
		if err != nil {
			return errors.WithStack(err)
		}
	}

	return nil
}
