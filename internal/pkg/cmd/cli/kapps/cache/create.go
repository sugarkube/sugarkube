package cache

import (
	"fmt"
	"github.com/spf13/cobra"
	"io"
)

type createCmd struct {
	out io.Writer
}

func newCreateCmd(out io.Writer) *cobra.Command {
	c := &createCmd{
		out: out,
	}

	cmd := &cobra.Command{
		Use:   "create [flags]",
		Short: fmt.Sprintf("Create kapp caches"),
		Long:  `Create a local kapps cache for a given manifest(s).`,
		RunE:  c.run,
	}

	return cmd
}

func (c *createCmd) run(cmd *cobra.Command, args []string) error {
	return nil
}
