package cluster

import (
	"fmt"
	"github.com/spf13/cobra"
	"io"
	"strings"
)

// Launches a cluster, either local or remote.

const createDesc = `
Create a new cluster.
`

type valueFiles []string

func (v *valueFiles) String() string {
	return fmt.Sprint(*v)
}

func (v *valueFiles) Type() string {
	return "valueFiles"
}

func (v *valueFiles) Set(value string) error {
	for _, filePath := range strings.Split(value, ",") {
		*v = append(*v, filePath)
	}
	return nil
}

type createCmd struct {
	provider   string
	valueFiles valueFiles
	profile    string
	account    string
	cluster    string
	region     string
}

func newCreateCmd(out io.Writer) *cobra.Command {

	t := &createCmd{}

	cmd := &cobra.Command{
		Use:   "create [flags]",
		Short: fmt.Sprintf("locally render templates"),
		Long:  createDesc,
		RunE:  t.run,
	}

	f := cmd.Flags()
	f.StringVarP(&t.provider, "provider", "p", "", "provider name")
	f.StringVarP(&t.profile, "profile", "l", "", "profile name")
	f.StringVarP(&t.account, "account", "a", "", "account name")
	f.StringVarP(&t.cluster, "cluster", "c", "", "cluster name")
	f.StringVarP(&t.region, "region", "r", "", "region name")
	f.VarP(&t.valueFiles, "values", "f", "specify values in a YAML file (can specify multiple)")
	return cmd
}

func (t *createCmd) run(cmd *cobra.Command, args []string) error {

	return nil
}
