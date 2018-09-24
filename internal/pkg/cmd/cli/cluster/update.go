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
	"github.com/imdario/mergo"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/sugarkube/sugarkube/internal/pkg/cmd"
	"github.com/sugarkube/sugarkube/internal/pkg/kapp"
	"github.com/sugarkube/sugarkube/internal/pkg/log"
	"github.com/sugarkube/sugarkube/internal/pkg/provider"
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
	manifests        cmd.Files
	onlineTimeout    uint32
	readyTimeout     uint32
}

func newUpdateCmd(out io.Writer) *cobra.Command {

	c := &updateCmd{
		out: out,
	}

	cmd := &cobra.Command{
		Use:   "update [flags]",
		Short: fmt.Sprintf("Update a cluster"),
		Long: `Update a cluster if supported by the provisioner.

If updating a named stack, just pass the stack name and path to the config file 
it's defined in, e.g.

	$ sugarkube cluster update --stack-name dev1 --stack-config /path/to/stacks.yaml

Otherwise specify the provider, profile, etc. on the command line, or to override
values in a stack config file. CLI args take precedence over values in stack 
config files.

Note: Not all providers require all arguments. See documentation for help.
`,
		RunE: c.run,
	}

	f := cmd.Flags()
	f.BoolVar(&c.dryRun, "dry-run", false, "show what would happen but don't update a cluster")
	f.StringVarP(&c.stackName, "stack-name", "n", "", "name of a stack to launch (required when passing --stack-config)")
	f.StringVarP(&c.stackFile, "stack-config", "s", "", "path to file defining stacks by name")
	f.StringVarP(&c.provider, "provider", "p", "", "name of provider, e.g. aws, local, etc.")
	f.StringVarP(&c.provisioner, "provisioner", "v", "", "name of provisioner, e.g. kops, minikube, etc.")
	f.StringVarP(&c.profile, "profile", "l", "", "launch profile, e.g. dev, test, prod, etc.")
	f.StringVarP(&c.cluster, "cluster", "c", "", "name of cluster to launch, e.g. dev1, dev2, etc.")
	f.StringVarP(&c.account, "account", "a", "", "string identifier for the account to launch in (for providers that support it)")
	f.StringVarP(&c.region, "region", "r", "", "name of region (for providers that support it)")
	f.VarP(&c.providerVarsDirs, "dir", "f", "Paths to YAML directory to load provider configs from (can specify multiple)")
	f.VarP(&c.manifests, "manifest", "m", "YAML manifest file to load (can specify multiple)")
	f.Uint32Var(&c.onlineTimeout, "online-timeout", 600, "max number of seconds to wait for the cluster to come online")
	f.Uint32Var(&c.readyTimeout, "ready-timeout", 600, "max number of seconds to wait for the cluster to become ready")
	return cmd
}

func (c *updateCmd) run(cmd *cobra.Command, args []string) error {

	stackConfig, err := ParseStackCliArgs(c.stackName, c.stackFile)
	if err != nil {
		return errors.WithStack(err)
	}

	cliManifests, err := kapp.ParseManifests(c.manifests)
	if err != nil {
		return errors.WithStack(err)
	}

	// CLI args override configured args, so merge them in
	cliStackConfig := &kapp.StackConfig{
		Provider:         c.provider,
		Provisioner:      c.provisioner,
		Profile:          c.profile,
		Cluster:          c.cluster,
		Region:           c.region,
		Account:          c.account,
		ProviderVarsDirs: c.providerVarsDirs,
		Manifests:        cliManifests,
		ReadyTimeout:     c.readyTimeout,
		OnlineTimeout:    c.onlineTimeout,
	}

	mergo.Merge(stackConfig, cliStackConfig, mergo.WithOverride)

	log.Debugf("Final stack config: %#v", stackConfig)

	providerImpl, err := provider.NewProvider(stackConfig)
	if err != nil {
		return errors.WithStack(err)
	}

	provisionerImpl, err := provisioner.NewProvisioner(stackConfig.Provisioner)
	if err != nil {
		return errors.WithStack(err)
	}

	online, err := provisioner.IsAlreadyOnline(provisionerImpl, stackConfig, providerImpl)
	if err != nil {
		return errors.WithStack(err)
	}

	if !online {
		log.Infof("Target cluster is not online. Aborting.")
		return nil
	}

	err = provisioner.Update(provisionerImpl, stackConfig, providerImpl, c.dryRun)
	if err != nil {
		return errors.WithStack(err)
	}

	if c.dryRun {
		log.Infof("Dry run. Skipping cluster readiness check.")
	} else {
		err = provisioner.WaitForClusterReadiness(provisionerImpl, stackConfig, providerImpl)
		if err != nil {
			return errors.WithStack(err)
		}

		log.Infof("Cluster '%s' is ready for use.", stackConfig.Cluster)
	}

	return nil
}
