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

package cluster

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/sugarkube/sugarkube/internal/pkg/printer"
	"github.com/sugarkube/sugarkube/internal/pkg/provisioner"
	"github.com/sugarkube/sugarkube/internal/pkg/stack"
	"github.com/sugarkube/sugarkube/internal/pkg/structs"
	"time"
)

// Launches a cluster, either local or remote.

type connectCmd struct {
	dryRun      bool
	stackName   string
	stackFile   string
	provider    string
	provisioner string
	profile     string
	account     string
	cluster     string
	region      string
}

func newConnectCmd() *cobra.Command {

	c := &connectCmd{}

	command := &cobra.Command{
		Use:   "connect [flags] [stack-file] [stack-name]",
		Short: fmt.Sprintf("Create a connection to a private Kubernetes API server"),
		Long: `Ensures an private Kubernetes API server is accessible from the local machine.
For kops clusters with a private API load balancer and a bastion SSH port forwarding
will be set up. 

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
	f.BoolVarP(&c.dryRun, "dry-run", "n", false, "show what would happen but don't connect a cluster")
	f.StringVar(&c.provider, "provider", "", "name of provider, e.g. aws, local, etc.")
	f.StringVar(&c.provisioner, "provisioner", "", "name of provisioner, e.g. kops, minikube, etc.")
	f.StringVar(&c.profile, "profile", "", "launch profile, e.g. dev, test, prod, etc.")
	f.StringVarP(&c.cluster, "cluster", "c", "", "name of cluster to launch, e.g. dev1, dev2, etc.")
	f.StringVarP(&c.account, "account", "a", "", "string identifier for the account to launch in (for providers that support it)")
	f.StringVarP(&c.region, "region", "r", "", "name of region (for providers that support it)")
	return command
}

func (c *connectCmd) run() error {

	// CLI overrides - will be merged with and take precedence over values loaded from the stack config file
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

	_, err = printer.Fprintf("Trying to connect to cluster '[bold]%s[reset]'...\n",
		stackObj.GetConfig().GetName())
	if err != nil {
		return errors.WithStack(err)
	}

	connected, err := provisioner.IsAlreadyOnline(stackObj.GetProvisioner(), c.dryRun)
	if err != nil {
		return errors.WithStack(err)
	}

	if connected {
		_, err = printer.Fprintf("[green]Connectivity established to the API server. Press " +
			"CTRL-C to quit.\n")
		if err != nil {
			return errors.WithStack(err)
		}

		for {
			time.Sleep(60 * time.Second)
		}
	} else {
		_, err = printer.Fprintln("[red]Connection failed. [reset]Gathering more information...")
		if err != nil {
			return errors.WithStack(err)
		}

		online, err := stackObj.GetProvisioner().IsAlreadyOnline(false)
		if err != nil {
			return errors.WithStack(err)
		}

		if online {
			_, err = printer.Fprintf("[red]Cluster '%s' [bold]is[reset][red] online. \n",
				stackObj.GetConfig().GetName())
			if err != nil {
				return errors.WithStack(err)
			}

			_, err = printer.Fprintln("[blue]Tip[reset]: This should have worked. You'll need to manually " +
				"investigate why we couldn't connect to a running cluster. Try rerunning this command with logging " +
				"enabled for more information.")
			if err != nil {
				return errors.WithStack(err)
			}
		} else {
			_, err = printer.Fprintf("[red]Cluster '%s' [bold]is not[reset][red] online\n",
				stackObj.GetConfig().GetName())
			if err != nil {
				return errors.WithStack(err)
			}

			_, err = printer.Fprintln("[blue]Tip[reset]: Create a cluster with the `[white]cluster create[reset]` " +
				"command or by using the `[white]cluster_update[reset]` pre-/post-action in a kapp.")
			if err != nil {
				return errors.WithStack(err)
			}
		}
	}

	return nil
}
