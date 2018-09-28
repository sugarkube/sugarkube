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
	"github.com/sugarkube/sugarkube/internal/pkg/cmd/cli/utils"
	"github.com/sugarkube/sugarkube/internal/pkg/kapp"
	"github.com/sugarkube/sugarkube/internal/pkg/log"
	"io"
	"io/ioutil"
)

type createCmd struct {
	out       io.Writer
	dryRun    bool
	stackName string
	stackFile string
	//manifests cmd.Files
	cacheDir string
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
	f.StringVarP(&c.stackName, "stack-name", "n", "", "name of a stack to launch (required when passing --stack-config)")
	f.StringVarP(&c.stackFile, "stack-config", "s", "", "path to file defining stacks by name")
	f.StringVarP(&c.cacheDir, "dir", "d", "", "Directory to build the cache in. A temp directory will be generated if not supplied.")
	// commented for now to keep things simple, but ultimately we should probably support taking these as CLI args
	//f.VarP(&c.manifests, "manifest", "m", "YAML manifest file to load (can specify multiple)")

	// todo - add a flag to automatically template kapps (or perhaps more usefully do that
	// by default and create a flag to prevent kapps being automatically templated)

	return cmd
}

func (c *createCmd) run(cmd *cobra.Command, args []string) error {

	log.Logger.Debugf("Got CLI args: %#v", c)

	stackConfig, err := utils.MaybeLoadStackConfig(c.stackName, c.stackFile)
	if err != nil {
		return errors.WithStack(err)
	}

	log.Logger.Debugf("Loaded stackConfig=%#v", stackConfig)

	//cliManifests, err := kapp.ParseManifests(c.manifests)
	//if err != nil {
	//	return errors.WithStack(err)
	//}

	// CLI args override configured args, so merge them in
	//cliStackConfig := &kapp.StackConfig{
	//	Manifests: cliManifests,
	//}
	//
	//mergo.Merge(stackConfig, cliStackConfig, mergo.WithOverride)

	log.Logger.Debugf("Final stack config: %#v", stackConfig)

	log.Logger.Debugf("Loaded %d manifest(s)", len(stackConfig.Manifests))

	for _, manifest := range stackConfig.Manifests {
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

	log.Logger.Debugf("Kapps validated. Caching manifests into %s...", cacheDir)

	for _, manifest := range stackConfig.Manifests {
		err := cacher.CacheManifest(manifest, cacheDir, c.dryRun)
		if err != nil {
			return errors.WithStack(err)
		}
	}

	log.Logger.Infof("Manifests cached to: %s", cacheDir)

	return nil
}
