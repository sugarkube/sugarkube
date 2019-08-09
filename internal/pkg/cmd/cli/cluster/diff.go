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

package cluster

import (
	"fmt"
	"github.com/spf13/cobra"
)

type diffCmd struct {
	extended bool
}

// Diff may not be the best term, since the output isn't only a diff but also
// a plan of changes that need to be applied against a target cluster. However,
// since we run kapps in a two-phased manner - first planning then applying
// changes - we won't use the more obvious term 'plan' here to avoid ambiguous
// terms.
func newDiffCmd() *cobra.Command {
	c := &diffCmd{}

	cmd := &cobra.Command{
		Use:   "diff [flags]",
		Short: fmt.Sprintf("Diff the state of a cluster with manifests"),
		Long: `Discovers the differences between the actual kapps installed on a cluster compared 
to the kapps that should be present/absent according to the manifests.

This command checks the current state of a cluster by consulting the configured 
Source-of-Truth. It compares that against the list of kapps specified in the 
manifests to be present or absent and then calculates which kapps should be 
installed and deleted.

When run with '--extended' this command will also include the contents of each
kapp's 'sugarkube.yaml' file (if it exists). This can be used to inform e.g.
a CI/CD system about the secrets that a kapp needs during installation.
`,
		RunE: c.run,
	}

	f := cmd.Flags()
	f.BoolVar(&c.extended, "extended", false, "include each kapp's 'sugarkube.yaml' file in output")

	return cmd
}

func (c *diffCmd) run(cmd *cobra.Command, args []string) error {
	// todo the diff should include a timestamp so that we can allow them to
	// only be valid as inputs to `kapps install` for a certain amount of time.

	return nil
}
