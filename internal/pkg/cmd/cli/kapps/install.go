package kapps

import (
	"fmt"
	"github.com/spf13/cobra"
	"io"
)

type installCmd struct {
	out io.Writer
}

func newInstallCmd(out io.Writer) *cobra.Command {
	c := &installCmd{
		out: out,
	}

	cmd := &cobra.Command{
		Use:   "install [flags]",
		Short: fmt.Sprintf("Install kapps"),
		Long:  `Install kapps from a manifest(s) into a target cluster.`,
		RunE:  c.run,
	}

	return cmd
}

func (c *installCmd) run(cmd *cobra.Command, args []string) error {
	return nil
}
