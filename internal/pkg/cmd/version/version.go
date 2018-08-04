package version

import (
	"fmt"
	"github.com/boosh/sugarkube/version"
	"github.com/spf13/cobra"
)

func NewCommand() *cobra.Command {
	c := &cobra.Command{
		Use:   "version",
		Short: "Print the version number of sugarkube",
		Long:  `All software has versions. This is sugarkube's.`,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("Build Date:", version.BuildDate)
			fmt.Println("Git Commit:", version.GitCommit)
			fmt.Println("Version:", version.Version)
			fmt.Println("Go Version:", version.GoVersion)
			fmt.Println("OS / Arch:", version.OsArch)
		},
	}

	return c
}
