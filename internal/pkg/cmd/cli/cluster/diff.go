package cluster

import (
	"fmt"
	"github.com/spf13/cobra"
	"io"
)

type diffCmd struct {
	out      io.Writer
	extended bool
}

func newDiffCmd(out io.Writer) *cobra.Command {
	c := &diffCmd{
		out: out,
	}

	cmd := &cobra.Command{
		Use:   "diff [flags]",
		Short: fmt.Sprintf("Diff the state of a cluster with manifests"),
		Long: `Discovers the differences between the actual kapps installed on a cluster compared 
to the kapps that should be present/absent according to the manifests.

This command checks the current state of a cluster by consulting the configured 
Source-of-Truth. It compares that against the list of kapps specified in the 
manifests to be present or absent and then calculates which kapps should be 
installed and destroyed.

When run with '--extended' this command will also include the contents of each
kapp's 'sugarkube.yaml' file (if it exists). This can be used to inform e.g.
a CI/CD system about the secrets that a kapp needs during installation.
`,
		RunE: c.run,
	}

	f := cmd.Flags()
	f.BoolVar(&c.extended, "extended", false, "include each kapp's 'sugarkube.yaml' file in output")

	return cmd
}

func (c *diffCmd) run(cmd *cobra.Command, args []string) error {
	return nil
}
