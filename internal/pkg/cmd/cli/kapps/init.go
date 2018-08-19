package kapps

import (
	"fmt"
	"github.com/spf13/cobra"
	"io"
)

type cmdConfig struct {
	out io.Writer
}

func newInitCmd(out io.Writer) *cobra.Command {
	c := &cmdConfig{
		out: out,
	}

	cmd := &cobra.Command{
		Use:   "init [flags]",
		Short: fmt.Sprintf("Initialise kapps"),
		Long: `Initialises kapps by generating necessary files, e.g. terraform backends
configured for the region the target cluster is in, generating Helm
'values.yaml' files, etc.`,
		RunE: c.run,
	}

	return cmd
}

func (c *cmdConfig) run(cmd *cobra.Command, args []string) error {
	return nil
}
