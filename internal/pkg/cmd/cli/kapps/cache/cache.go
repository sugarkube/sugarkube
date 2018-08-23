package cache

import (
	"fmt"
	"github.com/spf13/cobra"
	"io"
)

func NewCacheCmds(out io.Writer) *cobra.Command {

	cmd := &cobra.Command{
		Use:   "cache [command]",
		Short: fmt.Sprintf("Work with kapp caches"),
		Long:  `Create and refresh kapp caches`,
	}

	cmd.AddCommand(
		newCreateCmd(out),
		newRefreshCmd(out),
		newValidateCmd(out),
	)

	return cmd
}
