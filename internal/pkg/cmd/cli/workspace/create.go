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
	"github.com/sugarkube/sugarkube/internal/pkg/cmd/cli/kapps"
	"github.com/sugarkube/sugarkube/internal/pkg/constants"
	"github.com/sugarkube/sugarkube/internal/pkg/log"
	"github.com/sugarkube/sugarkube/internal/pkg/printer"
	"github.com/sugarkube/sugarkube/internal/pkg/stack"
	"github.com/sugarkube/sugarkube/internal/pkg/structs"
	"path/filepath"
)

type createCmd struct {
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
}

func newCreateCmd() *cobra.Command {
	c := &createCmd{}

	cmd := &cobra.Command{
		Use:   "create [flags] [stack-file] [stack-name] [workspace-dir]",
		Short: fmt.Sprintf("Create a workspace"),
		Long: `Create/update a local workspace for a given manifest(s), and renders any 
templates defined by kapps.`,
		RunE: func(cmd *cobra.Command, args []string) error {
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

	f := cmd.Flags()
	f.BoolVarP(&c.dryRun, "dry-run", "n", false, "show what would happen but don't create a cluster")
	f.BoolVarP(&c.renderTemplates, "template", "t", false, "render templates for kapps ignoring any errors")
	f.StringVar(&c.provider, "provider", "", "name of provider, e.g. aws, local, etc.")
	f.StringVar(&c.provisioner, "provisioner", "", "name of provisioner, e.g. kops, minikube, etc.")
	f.StringVar(&c.profile, "profile", "", "launch profile, e.g. dev, test, prod, etc.")
	f.StringVarP(&c.cluster, "cluster", "c", "", "name of cluster to launch, e.g. dev1, dev2, etc.")
	f.StringVarP(&c.account, "account", "a", "", "string identifier for the account to launch in (for providers that support it)")
	f.StringVarP(&c.region, "region", "r", "", "name of region (for providers that support it)")

	return cmd
}

func (c *createCmd) run() error {

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

	for _, manifest := range stackObj.GetConfig().Manifests() {
		err := cacher.CacheManifest(manifest, absRootWorkspaceDir, c.dryRun)
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
		dagObj, err := kapps.BuildDagForSelected(stackObj, c.workspaceDir, []string{}, []string{},
			false, "")
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