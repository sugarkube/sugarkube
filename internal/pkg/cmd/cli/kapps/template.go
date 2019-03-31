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
	"github.com/sugarkube/sugarkube/internal/pkg/config"
	"github.com/sugarkube/sugarkube/internal/pkg/constants"
	"github.com/sugarkube/sugarkube/internal/pkg/interfaces"
	"github.com/sugarkube/sugarkube/internal/pkg/log"
	"github.com/sugarkube/sugarkube/internal/pkg/stack"
	"github.com/sugarkube/sugarkube/internal/pkg/structs"
	"io"
	"strings"
)

type templateConfig struct {
	out             io.Writer
	dryRun          bool
	cacheDir        string
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

func newTemplateCmd(out io.Writer) *cobra.Command {
	c := &templateConfig{
		out: out,
	}

	cmd := &cobra.Command{
		Use:   "template [flags] [stack-file] [stack-name] [cache-dir]",
		Short: fmt.Sprintf("Render templates for kapps"),
		Long: `Renders configured templates for kapps, useful for e.g. terraform backends 
configured for the region the target cluster is in, generating Helm 
'values.yaml' files, etc.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) < 3 {
				return errors.New("some required arguments are missing")
			} else if len(args) > 3 {
				return errors.New("too many arguments supplied")
			}
			c.stackFile = args[0]
			c.stackName = args[1]
			c.cacheDir = args[2]
			return c.run()
		},
	}

	f := cmd.Flags()
	f.BoolVarP(&c.dryRun, "dry-run", "n", false, "show what would happen but don't create a cluster")
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
	return cmd
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

	stackObj, err := stack.BuildStack(c.stackName, c.stackFile, cliStackConfig, config.CurrentConfig, c.out)
	if err != nil {
		return errors.WithStack(err)
	}

	// selected kapps will be returned in the order in which they appear in manifests, not the order they're specified
	// in selectors
	selectedInstallables, err := stack.SelectInstallables(stackObj.GetConfig().Manifests(),
		c.includeSelector, c.excludeSelector)
	if err != nil {
		return errors.WithStack(err)
	}

	// load the configs for the selected installables
	err = stack.LoadInstallables(selectedInstallables, c.cacheDir)
	if err != nil {
		return errors.WithStack(err)
	}

	_, err = fmt.Fprintf(c.out, "Rendering templates for %d kapps\n", len(selectedInstallables))
	if err != nil {
		return errors.WithStack(err)
	}

	err = RenderTemplates(selectedInstallables, c.cacheDir, stackObj, c.dryRun)
	if err != nil {
		return errors.WithStack(err)
	}

	_, err = fmt.Fprintln(c.out, "Templates successfully rendered")
	if err != nil {
		return errors.WithStack(err)
	}

	return nil
}

// Render templates for kapps defined in a stack config
func RenderTemplates(installables []interfaces.IInstallable, cacheDir string, stack interfaces.IStack,
	dryRun bool) error {

	if len(installables) == 0 {
		return errors.New("No installables supplied to template function")
	}

	candidateKappIds := make([]string, 0)
	for _, k := range installables {
		candidateKappIds = append(candidateKappIds, k.FullyQualifiedId())
	}

	log.Logger.Debugf("Rendering templates for installables: %s", strings.Join(candidateKappIds, ", "))

	for _, kappObj := range installables {
		templatedVars, err := stack.GetTemplatedVars(kappObj, map[string]interface{}{})
		if err != nil {
			return errors.WithStack(err)
		}

		_, err = kappObj.RenderTemplates(templatedVars, stack.GetConfig(), dryRun)
		if err != nil {
			return errors.WithStack(err)
		}
	}

	return nil
}
