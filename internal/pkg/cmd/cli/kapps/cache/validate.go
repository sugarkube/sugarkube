package cache

import (
	"fmt"
	"github.com/spf13/cobra"
	"io"
)

type validateCmd struct {
	out io.Writer
}

func newValidateCmd(out io.Writer) *cobra.Command {
	c := &validateCmd{
		out: out,
	}

	cmd := &cobra.Command{
		Use:   "validate [flags]",
		Short: fmt.Sprintf("Validate that a kapp cache matches the given manifests"),
		Long: `Validates that a kapp cache directory faithfully represent the kapps defined
in a manifest(s). The manifests can either defined in a stack config file or as command line
arguments.

Validateing means:
  * Read all the kapps from the manifests
  * Verify that each kapp in the cache is checked out at the desired branch and
    (probably optionally) that there are no uncommitted changes.
`,
		RunE: c.run,
	}

	return cmd
}

func (c *validateCmd) run(cmd *cobra.Command, args []string) error {
	// todo - could this just be a flag on `cache create --validate`? That way
	// it could create or validate an existing cache which will simplify scripting

	return nil
}
