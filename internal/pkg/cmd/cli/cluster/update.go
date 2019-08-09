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
	"github.com/sugarkube/sugarkube/internal/pkg/interfaces"
	"github.com/sugarkube/sugarkube/internal/pkg/log"
	"github.com/sugarkube/sugarkube/internal/pkg/printer"
	"github.com/sugarkube/sugarkube/internal/pkg/provisioner"
	"github.com/sugarkube/sugarkube/internal/pkg/stack"
	"github.com/sugarkube/sugarkube/internal/pkg/structs"
)

// Update a cluster if supported by the provisioner

type updateCmd struct {
	dryRun        bool
	skipCreate    bool
	stackName     string
	stackFile     string
	provider      string
	provisioner   string
	profile       string
	account       string
	cluster       string
	region        string
	onlineTimeout uint32
	readyTimeout  uint32
}

func newUpdateCmd() *cobra.Command {

	c := &updateCmd{}

	command := &cobra.Command{
		Use:   "update [flags] [stack-file] [stack-name]",
		Short: fmt.Sprintf("Update a cluster"),
		Long: `Update a cluster if supported by the provisioner.

Update a configured cluster, e.g.:

	$ sugarkube cluster update /path/to/stacks.yaml dev1 

Certain values can be provided to override values from the stack config file, e.g. to change the 
region, etc. 

Note: Not all providers require all arguments. See documentation for help.
`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) < 2 {
				return errors.New("the name of the stack to run, and the path to the stack file are required")
			} else if len(args) > 2 {
				return errors.New("too many arguments supplied")
			}
			c.stackFile = args[0]
			c.stackName = args[1]
			return c.run()
		},
	}

	f := command.Flags()
	f.BoolVarP(&c.dryRun, "dry-run", "n", false, "show what would happen but don't create a cluster")
	f.BoolVar(&c.skipCreate, "no-create", false, "don't automatically create the target cluster if it's offline")
	f.StringVar(&c.provider, "provider", "", "name of provider, e.g. aws, local, etc.")
	f.StringVar(&c.provisioner, "provisioner", "", "name of provisioner, e.g. kops, minikube, etc.")
	f.StringVar(&c.profile, "profile", "", "launch profile, e.g. dev, test, prod, etc.")
	f.StringVarP(&c.cluster, "cluster", "c", "", "name of cluster to launch, e.g. dev1, dev2, etc.")
	f.StringVarP(&c.account, "account", "a", "", "string identifier for the account to launch in (for providers that support it)")
	f.StringVarP(&c.region, "region", "r", "", "name of region (for providers that support it)")
	f.Uint32Var(&c.onlineTimeout, "online-timeout", 600, "max number of seconds to wait for the cluster to come online")
	f.Uint32Var(&c.readyTimeout, "ready-timeout", 600, "max number of seconds to wait for the cluster to become ready")
	return command
}

func (c *updateCmd) run() error {

	// CLI overrides - will be merged with any loaded from a stack config file
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

	stackObj.GetConfig().SetReadyTimeout(c.readyTimeout)
	stackObj.GetConfig().SetOnlineTimeout(c.onlineTimeout)

	err = UpdateCluster(stackObj, !c.skipCreate, c.dryRun)
	if err != nil {
		return errors.WithStack(err)
	}

	return nil
}

// Updates a cluster with a stack config
func UpdateCluster(stackObj interfaces.IStack, autoCreate bool,
	dryRun bool) error {
	dryRunPrefix := ""
	if dryRun {
		dryRunPrefix = "[Dry run] "
	}

	_, err := printer.Fprintf("%sChecking whether the target cluster '%s' is already "+
		"online...\n", dryRunPrefix, stackObj.GetConfig().GetCluster())
	if err != nil {
		return errors.WithStack(err)
	}

	online, err := provisioner.IsAlreadyOnline(stackObj.GetProvisioner(), dryRun)
	if err != nil {
		return errors.WithStack(err)
	}

	if !online {
		if autoCreate {
			_, err = printer.Fprintf("%sCluster isn't online. Will create it...\n", dryRunPrefix)
			if err != nil {
				return errors.WithStack(err)
			}

			err = CreateCluster(stackObj, dryRun)
			if err != nil {
				return errors.WithStack(err)
			}

		} else {
			_, err = printer.Fprintf("%sCluster isn't online but we're not to create it. Aborting.\n", dryRunPrefix)
			if err != nil {
				return errors.WithStack(err)
			}
			return nil
		}
	} else {
		_, err = printer.Fprintf("%sCluster is online. Will update it now (this "+
			"may take some time)...\n", dryRunPrefix)
		if err != nil {
			return errors.WithStack(err)
		}

		err = stackObj.GetProvisioner().Update(dryRun)
		if err != nil {
			return errors.WithStack(err)
		}

		if dryRun {
			log.Logger.Infof("%sSkipping cluster readiness check.", dryRunPrefix)
		} else {
			err = provisioner.WaitForClusterReadiness(stackObj.GetProvisioner())
			if err != nil {
				return errors.WithStack(err)
			}

			_, err = printer.Fprintf("%sCluster '%s' successfully updated.\n",
				dryRunPrefix, stackObj.GetConfig().GetCluster())
			if err != nil {
				return errors.WithStack(err)
			}
		}
	}

	return nil
}
