package sugarkube

import (
	"github.com/spf13/cobra"
	"github.com/sugarkube/sugarkube/internal/pkg/cmd/cli/cluster"
	"github.com/sugarkube/sugarkube/internal/pkg/cmd/version"
)

func NewCommand(name string) *cobra.Command {

	cmd := &cobra.Command{
		Use:   name,
		Short: "Route-to-Live deployment manager",
		Long: `A longer description that spans multiple lines and likely contains
examples and usage of using your application. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
		// Uncomment the following line if your bare application
		// has an action associated with it:
		//      Run: func(cmd *cobra.Command, args []string) { },
	}

	out := cmd.OutOrStdout()

	cmd.AddCommand(
		version.NewCommand(),
		cluster.NewClusterCmds(out),
	)

	return cmd
}

//func init() {
//	cobra.OnInitialize()
//
//	// Cobra also supports local flags, which will only run
//	// when this action is called directly.
//	// rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
//}
