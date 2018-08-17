package cluster

import (
	"fmt"
	"github.com/spf13/cobra"
	"io"
)

func NewClusterCmds(out io.Writer) *cobra.Command {

	cmd := &cobra.Command{
		Use:   "cluster [command]",
		Short: fmt.Sprintf("Work with clusters"),
		Long:  `Work with clusters`,
	}

	cmd.AddCommand(
		newCreateCmd(out),
	)

	return cmd
}
