package kapps

import (
	"fmt"
	"github.com/imdario/mergo"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/sugarkube/sugarkube/internal/pkg/cmd"
	"github.com/sugarkube/sugarkube/internal/pkg/cmd/cli/cluster"
	"github.com/sugarkube/sugarkube/internal/pkg/kapp"
	"github.com/sugarkube/sugarkube/internal/pkg/log"
	"github.com/sugarkube/sugarkube/internal/pkg/plan"
	"io"
)

type installCmd struct {
	out           io.Writer
	diffPath      string
	cacheDir      string
	dryRun        bool
	apply         bool
	oneShot       bool
	stackName     string
	stackFile     string
	provider      string
	provisioner   string
	varsFilesDirs cmd.Files
	profile       string
	account       string
	cluster       string
	region        string
	manifests     cmd.Files
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
	f.BoolVar(&c.apply, "apply", false, "actually apply a cluster diff to install/destroy kapps. If false, kapps "+
		"will be expected to plan their changes but not make any destrucive changes (e.g. should run 'terraform plan', etc. but not "+
		"apply it).")
	f.BoolVar(&c.oneShot, "one-shot", false, "apply a cluster diff in a single pass by invoking each kapp with "+
		"'APPROVED=false' then 'APPROVED=true' to install/destroy kapps in a single invocation of sugarkube")
	// todo - in future, as a convenience, add a --diff flag to auto-generate a cluster diff prior to installation instead of requiring
	// sugarkube to be invoked multiple times (first to create the cluster diff, then to install kapps)
	f.StringVarP(&c.diffPath, "diff-path", "d", "", "Path to the cluster diff to install")
	f.StringVarP(&c.stackName, "stack-name", "n", "", "name of a stack to launch (required when passing --stack-config)")
	f.StringVarP(&c.stackFile, "stack-config", "s", "", "path to file defining stacks by name")
	f.StringVarP(&c.provider, "provider", "p", "", "name of provider, e.g. aws, local, etc.")
	f.StringVarP(&c.provisioner, "provisioner", "v", "", "name of provisioner, e.g. kops, minikube, etc.")
	f.StringVarP(&c.profile, "profile", "l", "", "launch profile, e.g. dev, test, prod, etc.")
	f.StringVarP(&c.cluster, "cluster", "c", "", "name of cluster to launch, e.g. dev1, dev2, etc.")
	f.StringVarP(&c.account, "account", "a", "", "string identifier for the account to launch in (for providers that support it)")
	f.StringVarP(&c.region, "region", "r", "", "name of region (for providers that support it)")
	f.VarP(&c.varsFilesDirs, "vars-file-or-dir", "f", "YAML vars file or directory to load (can specify multiple)")
	f.VarP(&c.manifests, "manifest", "m", "YAML manifest file to load (can specify multiple but will replace any configured in a stack)")
	return cmd
}

func (c *installCmd) run() error {

	stackConfig, err := cluster.ParseStackCliArgs(c.stackName, c.stackFile)
	if err != nil {
		return errors.WithStack(err)
	}

	cliManifests, err := kapp.ParseManifests(c.manifests)
	if err != nil {
		return errors.WithStack(err)
	}

	// CLI args override configured args, so merge them in
	cliStackConfig := &kapp.StackConfig{
		Provider:      c.provider,
		Provisioner:   c.provisioner,
		Profile:       c.profile,
		Cluster:       c.cluster,
		VarsFilesDirs: c.varsFilesDirs,
		Manifests:     cliManifests,
	}

	mergo.Merge(stackConfig, cliStackConfig, mergo.WithOverride)

	log.Debugf("Final stack config: %#v", stackConfig)

	// todo - validate that the cluster diff matches the manifests

	// todo - load the cluster diff which contains the list of kapps to
	// install/destroy and at which versions
	// todo - diff the cache against the kapps in the cluster diff and abort if
	// it's out-of-sync (unless flags are set to ignore cache changes)
	//cacheDiff, err := cacher.DiffKappCache(clusterDiff, c.cacheDir)
	//if err != nil {
	//	return errors.WithStack(err)
	//}
	//if len(diff) != 0 {
	//	return errors.New("Cache out-of-sync with manifests: %s", diff)
	//}

	// todo - accept a previously generated diff as a CLI arg. If given, load
	// it and validate that the embedded stack config matches the target cluster.

	// planning mode, so generate a plan
	//if !c.apply {
	changePlan, err := plan.Create(stackConfig, c.cacheDir)
	if err != nil {
		return errors.WithStack(err)
	}

	// todo - if autoApply continue, otherwise output a diff of which kapps
	// will be installed/destroyed and return
	//}

	// if not in planning mode, apply plan
	// todo - think of better names than plan, apply and approved. The difference
	// between applying and approving isn't obvious.
	changePlan.Apply(c.apply, c.dryRun)

	return nil
}
