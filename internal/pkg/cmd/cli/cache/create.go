/*
 * Copyright 2018 The Sugarkube Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package cache

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/sugarkube/sugarkube/internal/pkg/cacher"
	"github.com/sugarkube/sugarkube/internal/pkg/cmd/cli/kapps"
	"github.com/sugarkube/sugarkube/internal/pkg/cmd/cli/utils"
	"github.com/sugarkube/sugarkube/internal/pkg/kapp"
	"github.com/sugarkube/sugarkube/internal/pkg/log"
	"io"
	"path/filepath"
)

type createCmd struct {
	out            io.Writer
	dryRun         bool
	stackName      string
	stackFile      string
	cacheDir       string
	skipTemplating bool
}

func newCreateCmd(out io.Writer) *cobra.Command {
	c := &createCmd{
		out: out,
	}

	cmd := &cobra.Command{
		Use:   "create [flags] [stack-file] [stack-name] [cache-dir]",
		Short: fmt.Sprintf("Create kapp caches"),
		Long: `Create/update a local kapps cache for a given manifest(s), and renders any 
templates defined by kapps.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) < 3 {
				return errors.New("some required arguments are missing")
			}
			c.stackFile = args[0]
			c.stackName = args[1]
			c.cacheDir = args[2]
			return c.run()
		},
	}

	f := cmd.Flags()
	f.BoolVar(&c.dryRun, "dry-run", false, "show what would happen but don't create a cluster")
	f.BoolVar(&c.skipTemplating, "skip-templating", false, "don't render templates for kapps")

	return cmd
}

func (c *createCmd) run() error {

	log.Logger.Debugf("Got CLI args: %#v", c)

	// CLI args override configured args, so merge them in
	cliStackConfig := &kapp.StackConfig{}

	stackConfig, providerImpl, _, err := utils.ProcessCliArgs(c.stackName,
		c.stackFile, cliStackConfig, c.out)
	if err != nil {
		return errors.WithStack(err)
	}

	log.Logger.Debugf("Loaded %d init manifest(s) and %d manifest(s)",
		len(stackConfig.InitManifests), len(stackConfig.Manifests))

	// todo - why is this here? why don't we always validate manifests?
	for _, manifest := range stackConfig.AllManifests() {
		err = kapp.ValidateManifest(&manifest)
		if err != nil {
			return errors.WithStack(err)
		}
	}

	log.Logger.Debugf("Manifests validated.")

	absCacheDir, err := filepath.Abs(c.cacheDir)
	if err != nil {
		return errors.WithStack(err)
	}

	log.Logger.Debugf("Caching manifests into %s...", absCacheDir)

	// don't use the abs cache path here to keep the output simpler
	_, err = fmt.Fprintf(c.out, "Caching kapps into '%s'...\n", c.cacheDir)
	if err != nil {
		return errors.WithStack(err)
	}

	for _, manifest := range stackConfig.AllManifests() {
		err := cacher.CacheManifest(manifest, absCacheDir, c.dryRun)
		if err != nil {
			return errors.WithStack(err)
		}
	}

	_, err = fmt.Fprintln(c.out, "Kapps successfully cached")
	if err != nil {
		return errors.WithStack(err)
	}

	log.Logger.Infof("Manifests cached to: %s", absCacheDir)

	if !c.skipTemplating {
		_, err = fmt.Fprintln(c.out, "Rendering templates for kapps...")
		if err != nil {
			return errors.WithStack(err)
		}

		// template kapps
		candidateKapps := map[string]kapp.Kapp{}

		for _, manifest := range stackConfig.AllManifests() {
			for _, manifestKapp := range manifest.Kapps {
				candidateKapps[manifestKapp.FullyQualifiedId()] = manifestKapp
			}
		}

		err = kapps.RenderTemplates(candidateKapps, absCacheDir, stackConfig, providerImpl,
			c.dryRun)
		if err != nil {
			return errors.WithStack(err)
		}

		_, err = fmt.Fprintln(c.out, "Templates successfully rendered")
		if err != nil {
			return errors.WithStack(err)
		}
	} else {
		_, err = fmt.Fprintln(c.out, "Skipping rendering templates for kapps")
		if err != nil {
			return errors.WithStack(err)
		}
	}

	_, err = fmt.Fprintf(c.out, "Kapps successfully cached into '%s'\n", absCacheDir)
	if err != nil {
		return errors.WithStack(err)
	}

	return nil
}
