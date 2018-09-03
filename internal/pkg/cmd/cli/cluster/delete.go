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
	"io"
)

type deleteCmd struct {
	out       io.Writer
	confirmed bool
}

func newDeleteCmd(out io.Writer) *cobra.Command {
	c := &deleteCmd{
		out: out,
	}

	cmd := &cobra.Command{
		Use:   "delete [flags]",
		Short: fmt.Sprintf("Delete a cluster"),
		Long:  `Tear down a target cluster.`,
		RunE:  c.run,
	}

	return cmd
}

func (c *deleteCmd) run(cmd *cobra.Command, args []string) error {
	return nil
}
