package sugarkube

import (
	"github.com/spf13/cobra"
	"github.com/sugarkube/sugarkube/internal/pkg/cmd/cli/cluster"
	"github.com/sugarkube/sugarkube/internal/pkg/cmd/version"
)

type rootCmd struct {
	dryRun bool
}

func NewCommand(name string) *cobra.Command {

	c := &rootCmd{}

	cmd := &cobra.Command{
		Use:   name,
		Short: "Route-to-Live deployment manager",
		Long: `Sugarkube is dependency management for your infrastructure. 
While its focus is Kubernetes-based clusters, it can be used to deploy your
applications onto any scriptable backend.

Dependencies are declared in 'manifest' files which describe which version of
an application to install onto whichever backend, similar to a Python/pip
'requirements.txt' file,  NPM 'package.json' or Java 'pom.xml'.

Sugarkube can also create Kubernetes clusters on various backends
(e.g. AWS, local, etc.) using a variety of provisioners (e.g. Kops, Minikube).

Use Sugarkube to:

  * Create your infrastructure from scratch on multiple backends, for full 
    disaster recovery and reproducible environments.
  * Automate building ephemeral dev/test environments.
  * Push your applications through a release pipeline, developing locally or
    in an (ephemeral) dev cluster, testing on staging, then releasing to one or 
    multiple target prod clusters.

See https://sugarkube.io for more info and documentation.
`,
		// Uncomment the following line if your bare application
		// has an action associated with it:
		//      Run: func(cmd *cobra.Command, args []string) { },
	}

	out := cmd.OutOrStdout()

	cmd.AddCommand(
		version.NewCommand(),
		cluster.NewClusterCmds(out),
	)

	f := cmd.Flags()
	f.BoolVar(&c.dryRun, "dry-run", false, "show what would happen but don't perform any destructive actions")

	return cmd
}

//func init() {
//	cobra.OnInitialize()
//
//	// Cobra also supports local flags, which will only run
//	// when this action is called directly.
//	// rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
//}
