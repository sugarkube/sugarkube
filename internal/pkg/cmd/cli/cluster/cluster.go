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
		Long:  `Create and delete clusters`,
	}

	cmd.AddCommand(
		newCreateCmd(out),
		newDeleteCmd(out),
	)

	return cmd
}
