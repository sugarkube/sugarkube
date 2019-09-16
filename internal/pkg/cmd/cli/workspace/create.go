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

package workspace

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/sugarkube/sugarkube/internal/pkg/cacher"
	"github.com/sugarkube/sugarkube/internal/pkg/cmd"
	"github.com/sugarkube/sugarkube/internal/pkg/constants"
	"github.com/sugarkube/sugarkube/internal/pkg/log"
	"github.com/sugarkube/sugarkube/internal/pkg/plan"
	"github.com/sugarkube/sugarkube/internal/pkg/printer"
	"github.com/sugarkube/sugarkube/internal/pkg/stack"
	"github.com/sugarkube/sugarkube/internal/pkg/structs"
	"path/filepath"
)

type createCommand struct {
	dryRun          bool
	stackName       string
	stackFile       string
	provider        string
	provisioner     string
	profile         string
	account         string
	cluster         string
	region          string
	workspaceDir    string
	renderTemplates bool
	includeSelector []string
	excludeSelector []string
}

func newCreateCommand() *cobra.Command {
	c := &createCommand{}

	usage := "create [flags] [stack-file] [stack-name] [workspace-dir]"
	command := &cobra.Command{
		Use:   usage,
		Short: fmt.Sprintf("Create a workspace"),
		Long: `Create/update a local workspace for a given manifest(s), and renders any 
templates defined by kapps.`,
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
	}

	f := command.Flags()
	f.BoolVarP(&c.dryRun, "dry-run", "n", false, "show what would happen but don't create a cluster")
	f.BoolVarP(&c.renderTemplates, "template", "t", false, "render templates for kapps ignoring any errors")
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

func (c *createCommand) run() error {

	log.Logger.Debugf("Got CLI args: %#v", c)

	// CLI args override configured args, so merge them in
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

	log.Logger.Debugf("Loaded %d manifest(s)", len(stackObj.GetConfig().Manifests()))

	// todo - why is this here? why don't we always validate manifests?
	for _, manifest := range stackObj.GetConfig().Manifests() {
		err = stack.ValidateManifest(manifest)
		if err != nil {
			return errors.WithStack(err)
		}
	}

	log.Logger.Debugf("Manifests validated.")

	absRootWorkspaceDir, err := filepath.Abs(c.workspaceDir)
	if err != nil {
		return errors.WithStack(err)
	}

	log.Logger.Debugf("Creating workspace at %s...", absRootWorkspaceDir)

	// don't use the abs workspace path here to keep the output simpler
	_, err = printer.Fprintf("[yellow]Creating/updating workspace at '[bold]%s'...\n", c.workspaceDir)
	if err != nil {
		return errors.WithStack(err)
	}

	// selected kapps will be returned in the order in which they appear in manifests, not the order
	// they're specified in selectors
	selectedInstallables, err := stack.SelectInstallables(stackObj.GetConfig().Manifests(),
		c.includeSelector, c.excludeSelector)
	if err != nil {
		return errors.WithStack(err)
	}

	selectedInstallableIds := make([]string, 0)

	for _, installableObj := range selectedInstallables {
		selectedInstallableIds = append(selectedInstallableIds,
			installableObj.FullyQualifiedId())
	}

	for _, manifest := range stackObj.GetConfig().Manifests() {
		err := cacher.CacheManifest(manifest, absRootWorkspaceDir, selectedInstallableIds, c.dryRun)
		if err != nil {
			return errors.WithStack(err)
		}

		// reload each installable now its been cached so we can render templates
		for _, installableObj := range manifest.Installables() {
			err := installableObj.LoadConfigFile(absRootWorkspaceDir)
			if err != nil {
				return errors.WithStack(err)
			}
		}
	}

	_, err = printer.Fprintln("[green]Finished downloading kapps")
	if err != nil {
		return errors.WithStack(err)
	}

	if c.renderTemplates {
		_, err = printer.Fprintln("\nRendering templates for kapps...")
		if err != nil {
			return errors.WithStack(err)
		}

		// create a DAG to template all the kapps
		dagObj, err := plan.BuildDagForSelected(stackObj, c.workspaceDir, []string{}, []string{}, false)
		if err != nil {
			return errors.WithStack(err)
		}

		err = dagObj.Execute(constants.DagActionTemplate, stackObj, false, true, true,
			true, true, c.dryRun)
		if err != nil {
			return errors.WithStack(err)
		}

		_, err = printer.Fprintln("[green]Templates successfully rendered")
		if err != nil {
			return errors.WithStack(err)
		}
	} else {
		_, err = printer.Fprintln("Skipping rendering templates for kapps")
		if err != nil {
			return errors.WithStack(err)
		}
	}

	_, err = printer.Fprintf("\n[green]Workspace successfully created at '%s'\n", absRootWorkspaceDir)
	if err != nil {
		return errors.WithStack(err)
	}

	return nil
}
