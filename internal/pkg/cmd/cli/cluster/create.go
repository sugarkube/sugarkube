package cluster

import (
	"fmt"
	"github.com/spf13/cobra"
	"io"
	"strings"
)

// Launches a cluster, either local or remote.

type varsFiles []string

func (v *varsFiles) String() string {
	return fmt.Sprint(*v)
}

func (v *varsFiles) Type() string {
	return "varsFiles"
}

func (v *varsFiles) Set(value string) error {
	for _, filePath := range strings.Split(value, ",") {
		*v = append(*v, filePath)
	}
	return nil
}

type createCmd struct {
	out       io.Writer
	stack     string
	stackFile string
	provider  string
	varsFiles varsFiles
	profile   string
	account   string
	cluster   string
	region    string
}

func newCreateCmd(out io.Writer) *cobra.Command {

	t := &createCmd{
		out: out,
	}

	cmd := &cobra.Command{
		Use:   "create [flags]",
		Short: fmt.Sprintf("Create a cluster"),
		Long: `Create a new cluster, either local or remote.

If creating a named stack, just pass the stack name and path to the file it's
defined in, e.g.

	$ sugarkube cluster create --stack dev1 --stack-file /path/to/stacks.yaml

Otherwise specify the provider, profile, etc. on the command line.

Note: Not all providers require all arguments. See documentation for help.
`,
		RunE: t.run,
	}

	f := cmd.Flags()
	f.StringVarP(&t.stack, "stack", "n", "", "name of a stack to launch")
	f.StringVarP(&t.stackFile, "stack-file", "s", "", "path to file defining stacks (required when passing --stack)")
	f.StringVarP(&t.provider, "provider", "p", "", "name of provider, e.g. aws, local, etc.")
	f.StringVarP(&t.profile, "profile", "l", "", "launch profile, e.g. dev, test, prod, etc.")
	f.StringVarP(&t.cluster, "cluster", "c", "", "name of cluster to launch, e.g. dev1, dev2, etc.")
	f.StringVarP(&t.account, "account", "a", "", "string identifier for the account to launch in (for providers that support it)")
	f.StringVarP(&t.region, "region", "r", "", "name of region (for providers that support it)")
	f.VarP(&t.varsFiles, "vars-file", "f", "YAML vars files to load (can specify multiple)")
	return cmd
}

func (t *createCmd) run(cmd *cobra.Command, args []string) error {

	return nil
}
