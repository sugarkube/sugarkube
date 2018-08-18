package cluster

import (
	"fmt"
	"github.com/imdario/mergo"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/sugarkube/sugarkube/internal/pkg/log"
	"github.com/sugarkube/sugarkube/internal/pkg/provider"
	"github.com/sugarkube/sugarkube/internal/pkg/provisioner"
	"github.com/sugarkube/sugarkube/internal/pkg/vars"
	"io"
	"strings"
)

// Launches a cluster, either local or remote.

type files []string

func (v *files) String() string {
	return fmt.Sprint(*v)
}

func (v *files) Type() string {
	return "files"
}

func (v *files) Set(value string) error {
	for _, filePath := range strings.Split(value, ",") {
		*v = append(*v, filePath)
	}
	return nil
}

type createCmd struct {
	out           io.Writer
	dryRun        bool
	stackName     string
	stackFile     string
	provider      string
	provisioner   string
	varsFilesDirs files
	profile       string
	account       string
	cluster       string
	region        string
	manifests     files
}

func newCreateCmd(out io.Writer) *cobra.Command {

	c := &createCmd{
		out: out,
	}

	cmd := &cobra.Command{
		Use:   "create [flags]",
		Short: fmt.Sprintf("Create a cluster"),
		Long: `Create a new cluster, either local or remote.

If creating a named stackName, just pass the stackName name and path to the file it's
defined in, e.g.

	$ sugarkube cluster create --stackName dev1 --stackName-file /path/to/stacks.yaml

Otherwise specify the provider, profile, etc. on the command line.

Note: Not all providers require all arguments. See documentation for help.
`,
		RunE: c.run,
	}

	f := cmd.Flags()
	f.BoolVar(&c.dryRun, "dry-run", false, "show what would happen but don't create a cluster")
	f.StringVarP(&c.stackName, "stack-name", "n", "", "name of a stack to launch (required when passing --stack-file)")
	f.StringVarP(&c.stackFile, "stack-config", "s", "", "path to file defining stacks (required when passing --stack)")
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

func (c *createCmd) run(cmd *cobra.Command, args []string) error {

	stackConfig := &vars.StackConfig{}
	var err error

	// make sure both stack name and stack file are supplied if either are supplied
	if c.stackName != "" || c.stackFile != "" {
		if c.stackName == "" {
			return errors.New("A stack name is required when supplying the path to a stack config file.")
		}

		if c.stackFile == "" {
			return errors.New("A stack config file is required when supplying a stack name.")
		}

		stackConfig, err = vars.LoadStackConfig(c.stackName, c.stackFile)
		if err != nil {
			return errors.WithStack(err)
		}
	}

	// CLI args override configured args, so merge them in
	cliStackConfig := &vars.StackConfig{
		Provider:      c.provider,
		Provisioner:   c.provisioner,
		Profile:       c.profile,
		Cluster:       c.cluster,
		VarsFilesDirs: c.varsFilesDirs,
		Manifests:     c.manifests,
	}

	mergo.Merge(stackConfig, cliStackConfig, mergo.WithOverride)

	log.Debugf("Final stack config: %#v", stackConfig)

	providerImpl, err := provider.NewProvider(stackConfig.Provider)
	if err != nil {
		return errors.WithStack(err)
	}

	stackConfigVars, err := provider.StackConfigVars(providerImpl, stackConfig)
	if err != nil {
		log.Warn("Error loading stack config variables")
		return errors.WithStack(err)
	}
	log.Debugf("Provider returned vars: %#v", stackConfigVars)

	if len(stackConfigVars) == 0 {
		log.Fatal("No values loaded for stack")
		return errors.New("Failed to load values for stack")
	}

	provisionerImpl, err := provisioner.NewProvisioner(stackConfig.Provisioner)
	if err != nil {
		return errors.WithStack(err)
	}

	online, err := provisioner.IsAlreadyOnline(provisionerImpl, stackConfig, stackConfigVars)
	if err != nil {
		return errors.WithStack(err)
	}

	if online {
		log.Infof("Target cluster is already online. Aborting.")
		return nil
	}

	err = provisioner.Create(provisionerImpl, stackConfig, stackConfigVars, c.dryRun)
	if err != nil {
		return errors.WithStack(err)
	}

	return nil
}
