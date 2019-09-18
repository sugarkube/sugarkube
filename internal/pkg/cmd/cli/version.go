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
	"fmt"
	"github.com/spf13/cobra"
	"github.com/sugarkube/sugarkube/internal/pkg/version"
)

type versionConfig struct {
	concise bool
}

func newVersionCommand() *cobra.Command {
	c := &versionConfig{}

	command := &cobra.Command{
		Use:   "version",
		Short: "Print the version number of sugarkube",
		Long:  `All software has versions. This is sugarkube's.`,
		Run: func(command *cobra.Command, args []string) {
			if c.concise {
				fmt.Printf(version.Version)
			} else {
				fmt.Println("Build Date:", version.BuildDate)
				fmt.Println("Git Commit:", version.GitCommit)
				fmt.Println("Version:", version.Version)
				fmt.Println("Go Version:", version.GoVersion)
				fmt.Println("OS / Arch:", version.OsArch)
			}
		},
	}

	f := command.Flags()
	f.BoolVarP(&c.concise, "concise", "c", false, "only print the version")

	return command
}
