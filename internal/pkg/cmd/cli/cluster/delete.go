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

package cluster

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/sugarkube/sugarkube/internal/pkg/cmd"
	"github.com/sugarkube/sugarkube/internal/pkg/constants"
	"github.com/sugarkube/sugarkube/internal/pkg/printer"
	"github.com/sugarkube/sugarkube/internal/pkg/stack"
	"github.com/sugarkube/sugarkube/internal/pkg/structs"
)

type deleteCommand struct {
	dryRun      bool
	approved    bool
	stackName   string
	stackFile   string
	provider    string
	provisioner string
	profile     string
	account     string
	cluster     string
	region      string
}

func newDeleteCommand() *cobra.Command {
	c := &deleteCommand{}

	usage := "delete [flags] [stack-file] [stack-name]"
	command := &cobra.Command{
		Use:   usage,
		Short: fmt.Sprintf("Delete a cluster"),
		Long:  `Tear down a target cluster.`,
		RunE: func(command *cobra.Command, args []string) error {
			err := cmd.ValidateNumArgs(args, 2, usage)
			if err != nil {
				return errors.WithStack(err)
			}
			c.stackFile = args[0]
			c.stackName = args[1]
			return c.run()
		},
	}

	f := command.Flags()
	f.BoolVarP(&c.dryRun, "dry-run", "n", false, "show what would happen but don't create a cluster")
	f.BoolVarP(&c.approved, "yes", "y", false, "actually delete the cluster")
	f.StringVar(&c.provider, "provider", "", "name of provider, e.g. aws, local, etc.")
	f.StringVar(&c.provisioner, "provisioner", "", "name of provisioner, e.g. kops, minikube, etc.")
	f.StringVar(&c.profile, "profile", "", "launch profile, e.g. dev, test, prod, etc.")
	f.StringVarP(&c.cluster, "cluster", "c", "", "name of cluster to launch, e.g. dev1, dev2, etc.")
	f.StringVarP(&c.account, "account", "a", "", "string identifier for the account to launch in (for providers that support it)")
	f.StringVarP(&c.region, "region", "r", "", "name of region (for providers that support it)")
	return command
}

func (c *deleteCommand) run() error {
	// CLI overrides - will be merged with and take precedence over values loaded from the stack config file
	cliStackConfig := &structs.StackFile{
		Provider:    c.provider,
		Provisioner: c.provisioner,
		Profile:     c.profile,
		Cluster:     c.cluster,
		Region:      c.region,
		Account:     c.account,
	}

	stackObj, err := stack.BuildStack(c.stackName, c.stackFile, cliStackConfig)
	if err != nil {
		return errors.WithStack(err)
	}

	dryRunPrefix := ""
	if c.dryRun {
		dryRunPrefix = "[Dry run] "
	}

	_, err = printer.Fprintf("%s[yellow]Deleting cluster (this may take some time)...[reset]\n", dryRunPrefix)
	if err != nil {
		return errors.WithStack(err)
	}

	err = stackObj.GetProvisioner().Delete(c.approved, c.dryRun)
	if err != nil {
		return errors.WithStack(err)
	}

	if c.approved {
		_, err = printer.Fprintf("%s[green]Cluster '%s' successfully deleted.\n",
			dryRunPrefix, stackObj.GetConfig().GetCluster())
		if err != nil {
			return errors.WithStack(err)
		}
	} else {
		_, err = printer.Fprintf("[white][bold]Note: [reset]No destructive changes were made. To actually "+
			"delete the cluster, rerun this command passing [cyan][bold]--%s[reset]\n", constants.YesFlag)
		if err != nil {
			return errors.WithStack(err)
		}
	}

	return nil
}
