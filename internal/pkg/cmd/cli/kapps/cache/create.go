package cache

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/sugarkube/sugarkube/internal/pkg/cacher"
	"github.com/sugarkube/sugarkube/internal/pkg/cmd"
	"github.com/sugarkube/sugarkube/internal/pkg/kapp"
	"github.com/sugarkube/sugarkube/internal/pkg/log"
	"io"
	"io/ioutil"
)

type createCmd struct {
	out       io.Writer
	dryRun    bool
	manifests cmd.Files
	cacheDir  string
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
	f.BoolVar(&c.dryRun, "dry-run", false, "show what would happen but don't create a cluster")
	f.StringVarP(&c.cacheDir, "dir", "d", "", "Directory to build the cache in. A temp directory will be generated if not supplied.")
	f.VarP(&c.manifests, "manifest", "m", "YAML manifest file to load (can specify multiple)")

	return cmd
}

func (c *createCmd) run(cmd *cobra.Command, args []string) error {
	manifests, err := kapp.ParseManifests(c.manifests)
	if err != nil {
		return errors.WithStack(err)
	}

	log.Debugf("Loaded %d manifest(s)", len(manifests))

	for _, manifest := range manifests {
		err = kapp.ValidateManifest(&manifest)
		if err != nil {
			return errors.WithStack(err)
		}
	}

	cacheDir := c.cacheDir
	if cacheDir == "" {
		tempDir, err := ioutil.TempDir("", "sugarkube-cache-")
		if err != nil {
			return errors.WithStack(err)
		}
		cacheDir = tempDir
	}

	log.Debugf("Kapps validated. Caching manifests into %s...", cacheDir)

	for _, manifest := range manifests {
		err := cacher.CacheManifest(manifest, cacheDir, c.dryRun)
		if err != nil {
			return errors.WithStack(err)
		}
	}

	log.Infof("Manifests cached to: %s", cacheDir)

	return nil
}
