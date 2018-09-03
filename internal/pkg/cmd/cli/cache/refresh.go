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
	"github.com/spf13/cobra"
	"io"
)

type refreshCmd struct {
	out io.Writer
}

func newRefreshCmd(out io.Writer) *cobra.Command {
	c := &refreshCmd{
		out: out,
	}

	cmd := &cobra.Command{
		Use:   "refresh [flags]",
		Short: fmt.Sprintf("Refresh kapp caches"),
		Long: `Refresh an existing kapps cache. This could perhaps be merged into a single
'cache'' command with a flag '--refresh' or '--update' to run in an existing
cache directory. I'm not sure we need 2 separate commands that are so
similar.

Refreshing means:
  * Read all the kapps from the manifests
  * Do git sparse checkouts and build the cache
  * Add flags for dealing with edited kapps (ignore, abort, etc.) and filtering
    kapps vs just checking them all out.
`,
		RunE: c.run,
	}

	return cmd
}

func (c *refreshCmd) run(cmd *cobra.Command, args []string) error {
	// this may end up as a flag on `cache create`, e.g. `cache create --refresh`,
	// but for now we'll implement it here as a separate command before adding
	// that flag, if at all.

	return nil
}
