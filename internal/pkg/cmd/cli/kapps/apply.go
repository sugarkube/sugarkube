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
	"github.com/sugarkube/sugarkube/internal/pkg/cmd/cli/utils"
	"github.com/sugarkube/sugarkube/internal/pkg/kapp"
	"github.com/sugarkube/sugarkube/internal/pkg/log"
	"github.com/sugarkube/sugarkube/internal/pkg/plan"
	"io"
)

type applyCmd struct {
	out             io.Writer
	diffPath        string
	cacheDir        string
	dryRun          bool
	approved        bool
	oneShot         bool
	force           bool
	initManifests   bool
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

func newApplyCmd(out io.Writer) *cobra.Command {
	c := &applyCmd{
		out: out,
	}

	cmd := &cobra.Command{
		Use:   "apply [flags] [stack-file] [stack-name] [cache-dir]",
		Short: fmt.Sprintf("Install/destroy kapps into a cluster"),
		Long:  `Apply cached kapps to a target cluster according to manifests.`,
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
	f.BoolVar(&c.approved, "approved", false, "actually apply a cluster diff to install/destroy kapps. If false, kapps "+
		"will be expected to plan their changes but not make any destrucive changes (e.g. should run 'terraform plan', etc. but not "+
		"apply it).")
	f.BoolVar(&c.oneShot, "one-shot", false, "apply a cluster diff in a single pass by invoking each kapp with "+
		"'APPROVED=false' then 'APPROVED=true' to install/destroy kapps in a single invocation of sugarkube")
	f.BoolVar(&c.force, "force", false, "don't require a cluster diff, just blindly install/destroy all the kapps "+
		"defined in a manifest(s)/stack config, even if they're already present/absent in the target cluster")
	f.BoolVar(&c.initManifests, "init", false, "only apply init manifests. If false (default) only apply normal manifests.")
	f.StringVarP(&c.diffPath, "diff-path", "d", "", "Path to the cluster diff to apply. If not given, a "+
		"diff will be generated")
	f.StringVar(&c.provider, "provider", "", "name of provider, e.g. aws, local, etc.")
	f.StringVar(&c.provisioner, "provisioner", "", "name of provisioner, e.g. kops, minikube, etc.")
	f.StringVar(&c.profile, "profile", "", "launch profile, e.g. dev, test, prod, etc.")
	f.StringVarP(&c.cluster, "cluster", "c", "", "name of cluster to launch, e.g. dev1, dev2, etc.")
	f.StringVarP(&c.account, "account", "a", "", "string identifier for the account to launch in (for providers that support it)")
	f.StringVarP(&c.region, "region", "r", "", "name of region (for providers that support it)")
	f.StringArrayVarP(&c.includeSelector, "include", "i", []string{},
		fmt.Sprintf("only process specified kapps (can specify multiple, formatted manifest-id:kapp-id or 'manifest-id:%s' for all)",
			kapp.WILDCARD_CHARACTER))
	f.StringArrayVarP(&c.excludeSelector, "exclude", "x", []string{},
		fmt.Sprintf("exclude individual kapps (can specify multiple, formatted manifest-id:kapp-id or 'manifest-id:%s' for all)",
			kapp.WILDCARD_CHARACTER))
	return cmd
}

func (c *applyCmd) run() error {

	// CLI overrides - will be merged with any loaded from a stack config file
	cliStackConfig := &kapp.StackConfig{
		Provider:    c.provider,
		Provisioner: c.provisioner,
		Profile:     c.profile,
		Cluster:     c.cluster,
		Region:      c.region,
		Account:     c.account,
	}

	stackConfig, err := utils.BuildStackConfig(c.stackName, c.stackFile, cliStackConfig, c.out)
	if err != nil {
		return errors.WithStack(err)
	}

	var manifests []*kapp.Manifest

	if c.initManifests {
		manifests = stackConfig.InitManifests
		log.Logger.Debugf("Creating a plan just for init manifests")
	} else {
		manifests = stackConfig.Manifests
		log.Logger.Debugf("Creating a plan just for normal manifests")
	}

	var actionPlan *plan.Plan

	if !c.force {
		panic("Cluster diffing not implemented. Pass --force")

		if c.diffPath != "" {
			// todo load a cluster diff from a file

			// todo - validate that the embedded stack config matches the target cluster.

			// in future we may want to be able to work entirely from a cluster
			// diff, in which case it'd really be a plan for us
			if len(stackConfig.Manifests) > 0 {
				// todo - validate that the cluster diff matches the manifests, e.g. that
				// the versions of kapps in the manifests match the versions in the cluster
				// diff
			}
		} else {
			// todo - create a cluster diff based on stackConfig.Manifests
		}

		// todo - diff the cache against the kapps in the cluster diff and abort if
		// it's out-of-sync (unless flags are set to ignore cache changes), e.g.:
		//cacheDiff, err := cacher.DiffKappCache(clusterDiff, c.cacheDir)
		//if err != nil {
		//	return errors.WithStack(err)
		//}
		//if len(diff) != 0 {
		//	return errors.New("Cache out-of-sync with manifests: %s", diff)
		//}

		// todo - create an action plan from the validated cluster diff
		//actionPlan, err := plan.FromDiff(clusterDiff)

	} else {
		_, err = fmt.Fprintln(c.out, "Planning operations on kapps")
		if err != nil {
			return errors.WithStack(err)
		}

		// force mode, so no need to perform validation. Just create a plan
		actionPlan, err = plan.Create(stackConfig, manifests, c.cacheDir, c.includeSelector, c.excludeSelector)
		if err != nil {
			return errors.WithStack(err)
		}
	}

	if !c.oneShot {
		_, err = fmt.Fprintf(c.out, "Applying the plan with APPROVED=%#v...\n", c.approved)
		if err != nil {
			return errors.WithStack(err)
		}

		// run the plan either preparing or applying changes
		err := actionPlan.Run(c.approved, c.dryRun)
		if err != nil {
			return errors.WithStack(err)
		}
	} else {
		_, err = fmt.Fprintln(c.out, "Applying the plan in a single pass")
		if err != nil {
			return errors.WithStack(err)
		}

		_, err = fmt.Fprintln(c.out, "First applying the plan with APPROVED=false for "+
			"kapps to plan their changes...")
		if err != nil {
			return errors.WithStack(err)
		}

		// one-shot mode, so prepare and apply the plan straight away
		err = actionPlan.Run(false, c.dryRun)
		if err != nil {
			return errors.WithStack(err)
		}

		_, err = fmt.Fprintln(c.out, "Now running with APPROVED=true to actually apply changes...")
		if err != nil {
			return errors.WithStack(err)
		}

		err = actionPlan.Run(true, c.dryRun)
		if err != nil {
			return errors.WithStack(err)
		}
	}

	_, err = fmt.Fprintln(c.out, "Kapp change plan successfully applied")
	if err != nil {
		return errors.WithStack(err)
	}

	return nil
}
