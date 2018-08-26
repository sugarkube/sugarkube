package kapps

import (
	"fmt"
	"github.com/spf13/cobra"
	"io"
)

func NewKappsCmds(out io.Writer) *cobra.Command {

	cmd := &cobra.Command{
		Use:   "kapps [command]",
		Short: fmt.Sprintf("Work with kapps"),
		Long:  `Install and uninstall kapps`,
	}

	cmd.AddCommand(
		newInitCmd(out),
		newInstallCmd(out),
	)

	return cmd
}
