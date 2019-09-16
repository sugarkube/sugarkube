/*
 * Copyright 2019 The Sugarkube Authors
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

package cli

import (
	"github.com/spf13/cobra"
	"os"
)

func newCompletionsCommand() *cobra.Command {
	c := &cobra.Command{
		Use:   "completion",
		Short: "Generate bash completions for Sugarkube",
		Long: `To load completion run

. <(sugarkube completion)

To configure your bash shell to load completions for each session add to your bashrc

# ~/.bashrc or ~/.profile
. <(sugarkube completion)
`,
		RunE: func(command *cobra.Command, args []string) error {
			return rootCommand.GenBashCompletion(os.Stdout)
		},
	}

	c.Aliases = []string{"completions"}

	return c
}
