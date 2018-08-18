package cluster

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/sugarkube/sugarkube/internal/pkg/log"
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

	t := &createCmd{
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
		RunE: t.run,
	}

	f := cmd.Flags()
	f.StringVarP(&t.stackName, "stack-name", "n", "", "name of a stack to launch (required when passing --stack-file)")
	f.StringVarP(&t.stackFile, "stack-config", "s", "", "path to file defining stacks (required when passing --stack)")
	f.StringVarP(&t.provider, "provider", "p", "", "name of provider, e.g. aws, local, etc.")
	f.StringVarP(&t.provisioner, "provisioner", "v", "", "name of provisioner, e.g. kops, minikube, etc.")
	f.StringVarP(&t.profile, "profile", "l", "", "launch profile, e.g. dev, test, prod, etc.")
	f.StringVarP(&t.cluster, "cluster", "c", "", "name of cluster to launch, e.g. dev1, dev2, etc.")
	f.StringVarP(&t.account, "account", "a", "", "string identifier for the account to launch in (for providers that support it)")
	f.StringVarP(&t.region, "region", "r", "", "name of region (for providers that support it)")
	f.VarP(&t.varsFilesDirs, "vars-file-or-dir", "f", "YAML vars file or directory to load (can specify multiple)")
	f.VarP(&t.manifests, "manifest", "m", "YAML manifest file to load (can specify multiple)")
	return cmd
}

func (c *createCmd) run(cmd *cobra.Command, args []string) error {

	var (
		stackConfig *vars.StackConfig
		err         error
	)

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
	} else {
		stackConfig = &vars.StackConfig{
			Provider:      c.provider,
			Provisioner:   c.provisioner,
			Profile:       c.profile,
			Cluster:       c.cluster,
			VarsFilesDirs: c.varsFilesDirs,
			Manifests:     c.manifests,
		}
	}

	log.Debugf("Loaded stack config: %#v", stackConfig)

	return nil
}
