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
	"github.com/sugarkube/sugarkube/internal/pkg/cmd/cli/utils"
	"github.com/sugarkube/sugarkube/internal/pkg/kapp"
	"github.com/sugarkube/sugarkube/internal/pkg/log"
	"github.com/sugarkube/sugarkube/internal/pkg/provisioner"
	"io"
)

// Update a cluster if supported by the provisioner

type updateCmd struct {
	out              io.Writer
	dryRun           bool
	stackName        string
	stackFile        string
	provider         string
	provisioner      string
	providerVarsDirs cmd.Files
	profile          string
	account          string
	cluster          string
	region           string
	onlineTimeout    uint32
	readyTimeout     uint32
}

func newUpdateCmd(out io.Writer) *cobra.Command {

	c := &updateCmd{
		out: out,
	}

	command := &cobra.Command{
		Use:   "update [flags] [stack-file] [stack-name]",
		Short: fmt.Sprintf("Update a cluster"),
		Long: `Update a cluster if supported by the provisioner.

Update a configured cluster, e.g.:

	$ sugarkube cluster update --stack-name dev1 --stack-config /path/to/stacks.yaml

Certain values can be provided to override values from the stack config file, e.g. to change the 
region, etc. 

Note: Not all providers require all arguments. See documentation for help.
`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) < 2 {
				return errors.New("the name of the stack to run, and the path to the stack file are required")
			}
			c.stackFile = args[0]
			c.stackName = args[1]
			return c.run()
		},
	}

	f := command.Flags()
	f.BoolVar(&c.dryRun, "dry-run", false, "show what would happen but don't update a cluster")
	f.StringVar(&c.provider, "provider", "", "name of provider, e.g. aws, local, etc.")
	f.StringVar(&c.provisioner, "provisioner", "", "name of provisioner, e.g. kops, minikube, etc.")
	f.StringVar(&c.profile, "profile", "", "launch profile, e.g. dev, test, prod, etc.")
	f.StringVarP(&c.cluster, "cluster", "c", "", "name of cluster to launch, e.g. dev1, dev2, etc.")
	f.StringVarP(&c.account, "account", "a", "", "string identifier for the account to launch in (for providers that support it)")
	f.StringVarP(&c.region, "region", "r", "", "name of region (for providers that support it)")
	f.VarP(&c.providerVarsDirs, "dir", "f", "Paths to YAML directory to load provider configs from (can specify multiple)")
	f.Uint32Var(&c.onlineTimeout, "online-timeout", 600, "max number of seconds to wait for the cluster to come online")
	f.Uint32Var(&c.readyTimeout, "ready-timeout", 600, "max number of seconds to wait for the cluster to become ready")
	return command
}

func (c *updateCmd) run() error {

	// CLI overrides - will be merged with any loaded from a stack config file
	cliStackConfig := &kapp.StackConfig{
		Provider:         c.provider,
		Provisioner:      c.provisioner,
		Profile:          c.profile,
		Cluster:          c.cluster,
		Region:           c.region,
		Account:          c.account,
		ProviderVarsDirs: c.providerVarsDirs,
		ReadyTimeout:     c.readyTimeout,
		OnlineTimeout:    c.onlineTimeout,
	}

	stackConfig, err := utils.ProcessCliArgs(c.stackName, c.stackFile, cliStackConfig, c.out)
	if err != nil {
		return errors.WithStack(err)
	}

	_, err = fmt.Fprintf(c.out, "Checking whether the target cluster '%s' is already "+
		"online...\n", stackConfig.Cluster)
	if err != nil {
		return errors.WithStack(err)
	}

	provisionerImpl, err := provisioner.NewProvisioner(stackConfig.Provisioner)
	if err != nil {
		return errors.WithStack(err)
	}

	online, err := provisioner.IsAlreadyOnline(provisionerImpl, stackConfig)
	if err != nil {
		return errors.WithStack(err)
	}

	if !online {
		_, err = fmt.Fprintln(c.out, "Cluster is already online. Aborting.")
		if err != nil {
			return errors.WithStack(err)
		}
		return nil
	}

	_, err = fmt.Fprintln(c.out, "Cluster is online. Will update it now (this "+
		"may take some time...)")
	if err != nil {
		return errors.WithStack(err)
	}

	err = provisioner.Update(provisionerImpl, stackConfig, c.dryRun)
	if err != nil {
		return errors.WithStack(err)
	}

	if c.dryRun {
		log.Logger.Infof("Dry run. Skipping cluster readiness check.")
	} else {
		err = provisioner.WaitForClusterReadiness(provisionerImpl, stackConfig)
		if err != nil {
			return errors.WithStack(err)
		}

		_, err = fmt.Fprintf(c.out, "Cluster '%s' successfully updated.\n", stackConfig.Cluster)
		if err != nil {
			return errors.WithStack(err)
		}
	}

	return nil
}
