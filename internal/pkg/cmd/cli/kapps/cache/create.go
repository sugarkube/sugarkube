package cache

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/sugarkube/sugarkube/internal/pkg/cmd"
	"github.com/sugarkube/sugarkube/internal/pkg/kapp"
	"github.com/sugarkube/sugarkube/internal/pkg/log"
	"io"
)

type createCmd struct {
	out       io.Writer
	manifests cmd.Files
}

func newCreateCmd(out io.Writer) *cobra.Command {
	c := &createCmd{
		out: out,
	}

	cmd := &cobra.Command{
		Use:   "create [flags]",
		Short: fmt.Sprintf("Create kapp caches"),
		Long:  `Create a local kapps cache for a given manifest(s).`,
		RunE:  c.run,
	}

	f := cmd.Flags()
	f.VarP(&c.manifests, "manifest", "m", "YAML manifest file to load (can specify multiple)")

	return cmd
}

func (c *createCmd) run(cmd *cobra.Command, args []string) error {
	kapps, err := kapp.ParseManifests(c.manifests)
	if err != nil {
		return errors.WithStack(err)
	}

	log.Debugf("Loaded %d kapp(s)", len(kapps))

	return nil
}
