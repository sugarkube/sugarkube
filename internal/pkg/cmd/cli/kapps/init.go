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

package kapps

import (
	"fmt"
	"github.com/spf13/cobra"
	"io"
)

type cmdConfig struct {
	out io.Writer
}

func newInitCmd(out io.Writer) *cobra.Command {
	c := &cmdConfig{
		out: out,
	}

	cmd := &cobra.Command{
		Use:   "init [flags]",
		Short: fmt.Sprintf("Initialise kapps"),
		Long: `Initialises kapps by generating necessary files, e.g. terraform backends
configured for the region the target cluster is in, generating Helm
'values.yaml' files, etc.`,
		RunE: c.run,
	}

	return cmd
}

func (c *cmdConfig) run(cmd *cobra.Command, args []string) error {
	return nil
}
