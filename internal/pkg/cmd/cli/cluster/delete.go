package cluster

import (
	"fmt"
	"github.com/spf13/cobra"
	"io"
)

type deleteCmd struct {
	out       io.Writer
	confirmed bool
}

func newDeleteCmd(out io.Writer) *cobra.Command {
	c := &deleteCmd{
		out: out,
	}

	cmd := &cobra.Command{
		Use:   "delete [flags]",
		Short: fmt.Sprintf("Delete a cluster"),
		Long:  `Tear down a target cluster.`,
		RunE:  c.run,
	}

	return cmd
}

func (c *deleteCmd) run(cmd *cobra.Command, args []string) error {
	return nil
}
