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

package kapps

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/sugarkube/sugarkube/internal/pkg/config"
	"github.com/sugarkube/sugarkube/internal/pkg/constants"
	"github.com/sugarkube/sugarkube/internal/pkg/stack"
	"github.com/sugarkube/sugarkube/internal/pkg/structs"
	"io"
)

type deleteCmd struct {
	out io.Writer
	//diffPath string
	cacheDir string
	dryRun   bool
	approved bool
	oneShot  bool
	//force               bool
	skipTemplating      bool
	skipPostActions     bool
	establishConnection bool
	stackName           string
	stackFile           string
	provider            string
	provisioner         string
	profile             string
	account             string
	cluster             string
	region              string
	includeSelector     []string
	excludeSelector     []string
}

func newDeleteCmd(out io.Writer) *cobra.Command {
	c := &deleteCmd{
		out: out,
	}

	cmd := &cobra.Command{
		Use:   "delete [flags] [stack-file] [stack-name] [cache-dir]",
		Short: fmt.Sprintf("Delete kapps from a cluster"),
		Long: `Delete cached kapps from a target cluster according to manifests.

For Kubernetes clusters with a non-public API server, the provisioner may need 
to set up connectivity to make it accessible to Sugarkube (e.g. by setting up 
SSH port forwarding via a bastion). This happens automatically when a cluster 
is created or updated by Sugarkube, but if you're deleting individual kapps 
you may need to pass the '--connect' flag to make Sugarkube go through that
process before deleting the selected kapps.
`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) < 3 {
				return errors.New("some required arguments are missing")
			} else if len(args) > 3 {
				return errors.New("too many arguments supplied")
			}
			c.stackFile = args[0]
			c.stackName = args[1]
			c.cacheDir = args[2]

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

	f := cmd.Flags()
	f.BoolVarP(&c.dryRun, "dry-run", "n", false, "show what would happen but don't create a cluster")
	f.BoolVarP(&c.approved, "yes", "y", false, "actually delete kapps. If false, kapps will be expected to plan "+
		"their changes but not make any destrucive changes (e.g. should run 'terraform plan', etc. but not apply it).")
	f.BoolVar(&c.oneShot, "one-shot", false, "invoke each kapp with 'APPROVED=false' then "+
		"'APPROVED=true' to delete kapps in a single pass")
	//f.BoolVar(&c.force, "force", false, "don't require a cluster diff, just blindly install/delete all the kapps "+
	//	"defined in a manifest(s)/stack config, even if they're already present/absent in the target cluster")
	f.BoolVarP(&c.skipTemplating, "no-template", "t", false, "skip writing templates for kapps before deleting them")
	f.BoolVar(&c.skipPostActions, "no-post-actions", false, "skip running post actions in kapps - useful to quickly tear down a cluster")
	f.BoolVar(&c.establishConnection, "connect", false, "establish a connection to the API server if it's not publicly accessible")
	//f.StringVarP(&c.diffPath, "diff-path", "d", "", "Path to the cluster diff to delete. If not given, a "+
	//	"diff will be generated")
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

func (c *deleteCmd) run() error {

	// CLI overrides - will be merged with any loaded from a stack config file
	cliStackConfig := &structs.StackFile{
		Provider:    c.provider,
		Provisioner: c.provisioner,
		Profile:     c.profile,
		Cluster:     c.cluster,
		Region:      c.region,
		Account:     c.account,
	}

	var err error

	stackObj, err = stack.BuildStack(c.stackName, c.stackFile, cliStackConfig, config.CurrentConfig, c.out)
	if err != nil {
		return errors.WithStack(err)
	}

	dryRunPrefix := ""
	if c.dryRun {
		dryRunPrefix = "[Dry run] "
	}

	// uncomment this when cluster diffing has been implemented
	//if !c.force {
	//	panic("Cluster diffing not implemented. Pass --force")
	//
	//	if c.diffPath != "" {
	//		// todo load a cluster diff from a file
	//
	//		// todo - validate that the embedded stack config matches the target cluster.
	//
	//		// in future we may want to be able to work entirely from a cluster
	//		// diff, in which case it'd really be a plan for us
	//		if len(stackObj.GetConfig().Manifests()) > 0 {
	//			// todo - validate that the cluster diff matches the manifests, e.g. that
	//			// the versions of kapps in the manifests match the versions in the cluster
	//			// diff
	//		}
	//	} else {
	//		// todo - create a cluster diff based on stackConfig.Manifests
	//	}
	//
	//	// todo - diff the cache against the kapps in the cluster diff and abort if
	//	// it's out-of-sync (unless flags are set to ignore cache changes), e.g.:
	//	//cacheDiff, err := cacher.DiffKappCache(clusterDiff, c.cacheDir)
	//	//if err != nil {
	//	//	return errors.WithStack(err)
	//	//}
	//	//if len(diff) != 0 {
	//	//	return errors.New("Cache out-of-sync with manifests: %s", diff)
	//	//}
	//
	//	// todo - create an action plan from the validated cluster diff
	//	//actionPlan, err := plan.FromDiff(clusterDiff)
	//
	//} else {
	_, err = fmt.Fprintf(c.out, "%sPlanning operations on kapps\n", dryRunPrefix)
	if err != nil {
		return errors.WithStack(err)
	}

	dagObj, err := BuildDagForSelected(stackObj, c.cacheDir, c.includeSelector, c.excludeSelector, "")
	if err != nil {
		return errors.WithStack(err)
	}

	// todo - print out the DAG
	//}

	if c.establishConnection {
		err = establishConnection(c.dryRun, dryRunPrefix)
		if err != nil {
			return errors.WithStack(err)
		}
	}

	shouldPlan := false
	approved := false

	if c.oneShot {
		shouldPlan = true
		approved = true
	} else {
		if c.approved {
			approved = true
		} else {
			shouldPlan = true
		}
	}

	err = dagObj.Execute(constants.DagActionDelete, stackObj, shouldPlan, approved, c.dryRun)
	if err != nil {
		return errors.WithStack(err)
	}

	_, err = fmt.Fprintf(c.out, "%sKapp change plan successfully applied\n", dryRunPrefix)
	if err != nil {
		return errors.WithStack(err)
	}

	return nil
}
