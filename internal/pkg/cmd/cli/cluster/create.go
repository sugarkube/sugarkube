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
	stackName string
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

	// make sure both stack name and stack file are supplied if either are supplied
	if t.stackName != "" || t.stackFile != "" {
		if t.stackName == "" {
			return errors.New("A stack name is required when supplying the path to a stack file")
		}

		if t.stackFile == "" {
			return errors.New("A stack file is required when supplying a stack name")
		}

		stack, err := vars.LoadStack(t.stackName, t.stackFile)
		if err != nil {
			errors.WithStack(err)
		}

		log.Debugf("Loaded stack: %#v", stack)
	}

	return nil
}
