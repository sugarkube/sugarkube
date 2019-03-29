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
	cmd2 "github.com/sugarkube/sugarkube/internal/pkg/cmd"
	"github.com/sugarkube/sugarkube/internal/pkg/interfaces"
	"io"
)

var stackObj interfaces.IStack

func NewKappsCmds(out io.Writer) *cobra.Command {

	cmd := &cobra.Command{
		Use:   "kapps [command]",
		Short: fmt.Sprintf("Work with kapps"),
		Long:  `Install and uninstall kapps`,
		PersistentPostRun: func(cmd *cobra.Command, args []string) {
			if stackObj != nil {
				// todo - run this even if there was an error / Ctrl-C
				err := stackObj.GetProvisioner().Close()
				if err != nil {
					cmd2.CheckError(err)
				}
			}
		},
	}

	cmd.AddCommand(
		newTemplateCmd(out),
		newInstallCmd(out),
		newDeleteCmd(out),
		newVarsCmd(out),
	)

	cmd.Aliases = []string{"kapp"}

	return cmd
}
