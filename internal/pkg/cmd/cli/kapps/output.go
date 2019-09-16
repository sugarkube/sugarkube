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
	"github.com/sugarkube/sugarkube/internal/pkg/constants"
	"github.com/sugarkube/sugarkube/internal/pkg/plan"
	"github.com/sugarkube/sugarkube/internal/pkg/printer"
	"github.com/sugarkube/sugarkube/internal/pkg/stack"
	"github.com/sugarkube/sugarkube/internal/pkg/structs"
)

type outputCommand struct {
	workspaceDir    string
	dryRun          bool
	includeParents  bool
	stackName       string
	stackFile       string
	provider        string
	provisioner     string
	profile         string
	account         string
	cluster         string
	region          string
	includeSelector []string
	excludeSelector []string
}

func newOutputCommand() *cobra.Command {
	c := &outputCommand{}

	command := &cobra.Command{
		Use:   "output [flags] [stack-file] [stack-name] [workspace-dir]",
		Short: fmt.Sprintf("Generate output for kapps"),
		Long:  "Makes all selected kapps generate output.\n",
		RunE: func(command *cobra.Command, args []string) error {
			if len(args) < 3 {
				return errors.New("some required arguments are missing")
			} else if len(args) > 3 {
				return errors.New("too many arguments supplied")
			}
			c.stackFile = args[0]
			c.stackName = args[1]
			c.workspaceDir = args[2]

			return c.run()
		},
	}

	f := command.Flags()
	f.BoolVarP(&c.dryRun, "dry-run", "n", false, "show what would happen but don't create a cluster")
	f.BoolVar(&c.includeParents, "parents", false, "process all parents of all selected kapps as well")
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
	return command
}

func (c *outputCommand) run() error {

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

	err = dagObj.Execute(constants.DagActionOutput, stackObj, false, true, true,
		true, false, c.dryRun)
	if err != nil {
		return errors.WithStack(err)
	}

	_, err = printer.Fprintf("%s[green]Kapps successfully processed\n", dryRunPrefix)
	if err != nil {
		return errors.WithStack(err)
	}

	return nil
}
