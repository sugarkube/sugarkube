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
	"io"
)

type installCmd struct {
	out           io.Writer
	dryRun        bool
	cacheDir      string
	stackName     string
	stackFile     string
	provider      string
	provisioner   string
	varsFilesDirs cmd.Files
	profile       string
	account       string
	cluster       string
	region        string
	// todo - add a command to validate that the cache matches the desired
	// state as defined in the manifest(s).
	manifests cmd.Files
	// todo - document that the above will replace manifests declared in a
	// stack config, not be additive

	// todo - add options to :
	// * create the cache into a target directory
	// * refresh the cache (i.e. modify it so it matches the manifests if it fails validation)
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
				return errors.New("the path to the kapp cache dir to install from is required")
			}
			c.cacheDir = args[0]
			return c.run()
		},
	}

	f := cmd.Flags()
	f.BoolVar(&c.dryRun, "dry-run", false, "show what would happen but don't create a cluster")
	f.StringVarP(&c.cacheDir, "dir", "d", "", "Cache directory to install kapps from")
	f.StringVarP(&c.stackName, "stack-name", "n", "", "name of a stack to launch (required when passing --stack-config)")
	f.StringVarP(&c.stackFile, "stack-config", "s", "", "path to file defining stacks by name")
	f.StringVarP(&c.provider, "provider", "p", "", "name of provider, e.g. aws, local, etc.")
	f.StringVarP(&c.provisioner, "provisioner", "v", "", "name of provisioner, e.g. kops, minikube, etc.")
	f.StringVarP(&c.profile, "profile", "l", "", "launch profile, e.g. dev, test, prod, etc.")
	f.StringVarP(&c.cluster, "cluster", "c", "", "name of cluster to launch, e.g. dev1, dev2, etc.")
	f.StringVarP(&c.account, "account", "a", "", "string identifier for the account to launch in (for providers that support it)")
	f.StringVarP(&c.region, "region", "r", "", "name of region (for providers that support it)")
	f.VarP(&c.varsFilesDirs, "vars-file-or-dir", "f", "YAML vars file or directory to load (can specify multiple)")
	f.VarP(&c.manifests, "manifest", "m", "YAML manifest file to load (can specify multiple)")
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

	// todo - validate the cache dir. Abort if the cache is out-of-sync with the manifests

	// todo - process the kapps. Run each manifest sequentially, but each
	// kapp in each manifest in parallel.

	return nil
}
