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

// Launches a cluster, either local or remote.

type createCmd struct {
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

func newCreateCmd(out io.Writer) *cobra.Command {

	c := &createCmd{
		out: out,
	}

	command := &cobra.Command{
		Use:   "create [flags] [stack-file] [stack-name]",
		Short: fmt.Sprintf("Create a cluster"),
		Long: `Create a new cluster, either local or remote.

Create a configured cluster, e.g.:

	$ sugarkube cluster create /path/to/stacks.yaml dev1

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
	f.StringVar(&c.provider, "provider", "", "name of provider, e.g. aws, local, etc.")
	f.StringVar(&c.provisioner, "provisioner", "", "name of provisioner, e.g. kops, minikube, etc.")
	f.StringVar(&c.profile, "profile", "", "launch profile, e.g. dev, test, prod, etc.")
	f.StringVarP(&c.cluster, "cluster", "c", "", "name of cluster to launch, e.g. dev1, dev2, etc.")
	f.StringVarP(&c.account, "account", "a", "", "string identifier for the account to launch in (for providers that support it)")
	f.StringVarP(&c.region, "region", "r", "", "name of region (for providers that support it)")
	f.VarP(&c.providerVarsDirs, "provider-dir", "f", "Paths to YAML directory to load provider configs from (can specify multiple times)")
	f.Uint32Var(&c.onlineTimeout, "online-timeout", 600, "max number of seconds to wait for the cluster to come online")
	f.Uint32Var(&c.readyTimeout, "ready-timeout", 600, "max number of seconds to wait for the cluster to become ready")
	return command
}

func (c *createCmd) run() error {

	// CLI overrides - will be merged with and take precedence over values loaded from the stack config file
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

	stackConfig, err := utils.BuildStackConfig(c.stackName, c.stackFile, cliStackConfig, c.out)
	if err != nil {
		return errors.WithStack(err)
	}

	err = CreateCluster(c.out, stackConfig, c.dryRun)
	if err != nil {
		return errors.WithStack(err)
	}

	return nil
}

// Creates a cluster with a stack config
func CreateCluster(out io.Writer, stackConfig *kapp.StackConfig, dryRun bool) error {
	_, err := fmt.Fprintf(out, "Checking whether the target cluster '%s' is already "+
		"online...\n", stackConfig.Cluster)
	if err != nil {
		return errors.WithStack(err)
	}

	provisionerImpl, err := provisioner.NewProvisioner(stackConfig.Provisioner, stackConfig)
	if err != nil {
		return errors.WithStack(err)
	}

	if dryRun {
		log.Logger.Infof("Dry run. Won't check if the cluster is already online.")

	} else {
		online, err := provisioner.IsAlreadyOnline(provisionerImpl, stackConfig)
		if err != nil {
			return errors.WithStack(err)
		}

		if online {
			_, err = fmt.Fprintln(out, "Cluster is already online. Aborting.")
			if err != nil {
				return errors.WithStack(err)
			}

			return nil
		}
	}

	_, err = fmt.Fprintln(out, "Cluster is not online. Will create it now (this "+
		"may take some time...)")
	if err != nil {
		return errors.WithStack(err)
	}

	err = provisioner.Create(provisionerImpl, stackConfig, dryRun)
	if err != nil {
		return errors.WithStack(err)
	}

	if dryRun {
		log.Logger.Infof("Dry run. Skipping cluster readiness check.")
	} else {
		err = provisioner.WaitForClusterReadiness(provisionerImpl, stackConfig)
		if err != nil {
			return errors.WithStack(err)
		}

		_, err = fmt.Fprintf(out, "Cluster '%s' successfully created.\n", stackConfig.Cluster)
		if err != nil {
			return errors.WithStack(err)
		}
	}

	return nil
}
