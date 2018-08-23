package kapps

import (
	"fmt"
	"github.com/spf13/cobra"
	"github.com/sugarkube/sugarkube/internal/pkg/cmd/cli/kapps/cache"
	"io"
)

func NewKappsCmds(out io.Writer) *cobra.Command {

	cmd := &cobra.Command{
		Use:   "kapps [command]",
		Short: fmt.Sprintf("Work with kapps"),
		Long:  `Install and uninstall kapps`,
	}

	cmd.AddCommand(
		cache.NewCacheCmds(out),
		newInitCmd(out),
		newInstallCmd(out),
	)

	return cmd
}
