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
	"github.com/sugarkube/sugarkube/internal/pkg/cmd"
	"github.com/sugarkube/sugarkube/internal/pkg/constants"
	"github.com/sugarkube/sugarkube/internal/pkg/plan"
	"github.com/sugarkube/sugarkube/internal/pkg/printer"
	"github.com/sugarkube/sugarkube/internal/pkg/stack"
	"github.com/sugarkube/sugarkube/internal/pkg/structs"
)

type templateConfig struct {
	dryRun          bool
	includeParents  bool
	ignoreErrors    bool
	workspaceDir    string
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

func newTemplateCommand() *cobra.Command {
	c := &templateConfig{}

	usage := "template [flags] [stack-file] [stack-name] [workspace-dir]"
	command := &cobra.Command{
		Use:   usage,
		Short: fmt.Sprintf("Render templates for kapps"),
		Long: `Renders configured templates for kapps, useful for e.g. terraform backends 
configured for the region the target cluster is in, generating Helm 
'values.yaml' files, etc.`,
		RunE: func(command *cobra.Command, args []string) error {
			err := cmd.ValidateNumArgs(args, 3, usage)
			if err != nil {
				return errors.WithStack(err)
			}
			c.stackFile = args[0]
			c.stackName = args[1]
			c.workspaceDir = args[2]
			return c.run()
		},
		Aliases: []string{"templates"},
	}

	f := command.Flags()
	f.BoolVarP(&c.dryRun, "dry-run", "n", false, "show what would happen but don't create a cluster")
	f.BoolVar(&c.includeParents, "parents", false, "process all parents of all selected kapps as well")
	f.BoolVar(&c.ignoreErrors, "ignore-errors", false, "ignore errors templating kapps")
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

func (c *templateConfig) run() error {

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

	// create a DAG to template all the kapps
	dagObj, err := plan.BuildDagForSelected(stackObj, c.workspaceDir, c.includeSelector, c.excludeSelector, c.includeParents)
	if err != nil {
		return errors.WithStack(err)
	}

	_, err = printer.Fprintln("")
	if err != nil {
		return errors.WithStack(err)
	}

	err = dagObj.Execute(constants.DagActionTemplate, stackObj, false, true, true,
		true, c.ignoreErrors, c.dryRun)
	if err != nil {
		return errors.WithStack(err)
	}

	_, err = printer.Fprintln("[green]Templates successfully rendered")
	if err != nil {
		return errors.WithStack(err)
	}

	return nil
}
