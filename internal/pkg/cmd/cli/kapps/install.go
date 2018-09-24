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
	"github.com/imdario/mergo"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/sugarkube/sugarkube/internal/pkg/cmd/cli/cluster"
	"github.com/sugarkube/sugarkube/internal/pkg/kapp"
	"github.com/sugarkube/sugarkube/internal/pkg/log"
	"github.com/sugarkube/sugarkube/internal/pkg/plan"
	"io"
)

type installCmd struct {
	out         io.Writer
	diffPath    string
	cacheDir    string
	dryRun      bool
	approved    bool
	oneShot     bool
	force       bool
	stackName   string
	stackFile   string
	provider    string
	provisioner string
	//kappVarsDirs cmd.Files
	profile string
	account string
	cluster string
	region  string
	//manifests    cmd.Files
	// todo - add options to :
	// * filter the kapps to be processed (use strings like e.g. manifest:kapp-id to refer to kapps)
	// * exclude manifests / kapps from being processed
}

func newInstallCmd(out io.Writer) *cobra.Command {
	c := &installCmd{
		out: out,
	}

	cmd := &cobra.Command{
		Use:   "install [cache-dir]",
		Short: fmt.Sprintf("Install kapps"),
		Long:  `Install cached kapps into a target cluster.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return errors.New("the path to the kapp cache dir is required")
			}
			c.cacheDir = args[0]
			return c.run()
		},
	}

	f := cmd.Flags()
	f.BoolVar(&c.dryRun, "dry-run", false, "show what would happen but don't create a cluster")
	f.BoolVar(&c.approved, "approved", false, "actually apply a cluster diff to install/destroy kapps. If false, kapps "+
		"will be expected to plan their changes but not make any destrucive changes (e.g. should run 'terraform plan', etc. but not "+
		"apply it).")
	f.BoolVar(&c.oneShot, "one-shot", false, "apply a cluster diff in a single pass by invoking each kapp with "+
		"'APPROVED=false' then 'APPROVED=true' to install/destroy kapps in a single invocation of sugarkube")
	f.BoolVar(&c.force, "force", false, "don't require a cluster diff, just blindly install/destroy all the kapps "+
		"defined in a manifest(s)/stack config, even if they're already present/absent in the target cluster")
	f.StringVarP(&c.diffPath, "diff-path", "d", "", "Path to the cluster diff to apply. If not given, a "+
		"diff will be generated")
	f.StringVarP(&c.stackName, "stack-name", "n", "", "name of a stack to launch (required when passing --stack-config)")
	f.StringVarP(&c.stackFile, "stack-config", "s", "", "path to file defining stacks by name")
	f.StringVarP(&c.provider, "provider", "p", "", "name of provider, e.g. aws, local, etc.")
	f.StringVarP(&c.provisioner, "provisioner", "v", "", "name of provisioner, e.g. kops, minikube, etc.")
	f.StringVarP(&c.profile, "profile", "l", "", "launch profile, e.g. dev, test, prod, etc.")
	f.StringVarP(&c.cluster, "cluster", "c", "", "name of cluster to launch, e.g. dev1, dev2, etc.")
	f.StringVarP(&c.account, "account", "a", "", "string identifier for the account to launch in (for providers that support it)")
	f.StringVarP(&c.region, "region", "r", "", "name of region (for providers that support it)")
	// these are commented for now to keep things simple, but ultimately we should probably support taking these as CLI args
	//f.VarP(&c.kappVarsDirs, "dir", "f", "Paths to YAML directory to load kapp values from (can specify multiple)")
	//f.VarP(&c.manifests, "manifest", "m", "YAML manifest file to load (can specify multiple but will replace any configured in a stack)")
	return cmd
}

func (c *installCmd) run() error {

	var err error

	stackConfig, err := cluster.ParseStackCliArgs(c.stackName, c.stackFile)
	if err != nil {
		return errors.WithStack(err)
	}

	//cliManifests, err := kapp.ParseManifests(c.manifests)
	//if err != nil {
	//	return errors.WithStack(err)
	//}

	// CLI args override configured args, so merge them in
	cliStackConfig := &kapp.StackConfig{
		Provider:    c.provider,
		Provisioner: c.provisioner,
		Profile:     c.profile,
		Cluster:     c.cluster,
		Region:      c.region,
		Account:     c.account,
		//KappVarsDirs: c.kappVarsDirs,
		//Manifests:    cliManifests,
	}

	mergo.Merge(stackConfig, cliStackConfig, mergo.WithOverride)

	log.Logger.Debugf("Final stack config: %#v", stackConfig)

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
		// force mode, so no need to perform validation. Just create a plan
		actionPlan, err = plan.Create(stackConfig, c.cacheDir)
		if err != nil {
			return errors.WithStack(err)
		}
	}

	if !c.oneShot {
		// run the plan either preparing or applying changes
		err := actionPlan.Run(c.approved, c.dryRun)
		if err != nil {
			return errors.WithStack(err)
		}
	} else {
		// one-shot mode, so prepare and apply the plan straight away
		err = actionPlan.Run(false, c.dryRun)
		if err != nil {
			return errors.WithStack(err)
		}
		err = actionPlan.Run(true, c.dryRun)
		if err != nil {
			return errors.WithStack(err)
		}
	}

	return nil
}
