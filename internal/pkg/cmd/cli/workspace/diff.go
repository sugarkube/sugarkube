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

package workspace

//import (
//	"fmt"
//	"github.com/spf13/cobra"
//	"io"
//)
//
//type diffCommand struct {
//}
//
//func newDiffCommand() *cobra.Command {
//	c := &diffCommand{}
//
//	command := &cobra.Command{
//		Use:   "diff [flags]",
//		Short: fmt.Sprintf("Diff a local kapp workspace against manifests"),
//		Long: `Diffs a local kapp workspace directory against kapps defined in a
//manifest(s). This is the difference between the current/actual state of the workspace
//vs the desired state. This command will print out any differences such as:
//  * The workspace containing kapps checked out at different versions to the those specified
//    in manifests
//  * Any changed/modified files in any kapps (as reported by the acquirer)
//
//The manifests can either defined in a stack config file or as command line
//arguments.
//`,
//		RunE: c.run,
//	}
//
//	return command
//}
//
//func (c *diffCommand) run(command *cobra.Command, args []string) error {
//	panic("Not implemented")
//}
