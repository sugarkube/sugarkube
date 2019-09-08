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
	"github.com/sugarkube/sugarkube/internal/pkg/interfaces"
	"github.com/sugarkube/sugarkube/internal/pkg/log"
	"os"
	"os/signal"
	"syscall"
)

var stackObj interfaces.IStack

func NewKappsCmds() *cobra.Command {

	cmd := &cobra.Command{
		Use:   "kapps [command]",
		Short: fmt.Sprintf("Work with kapps"),
		Long:  `Install and uninstall kapps`,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			log.Logger.Debug("Setting up signal handler")
			// catch termination via CTRL-C
			signals := make(chan os.Signal)
			signal.Notify(signals, os.Interrupt, syscall.SIGTERM, syscall.SIGKILL)
			go func() {
				<-signals
				log.Logger.Info("Caught termination signal. Will try to gracefully terminate...")
				if stackObj != nil {
					err2 := stackObj.GetProvisioner().Close()
					if err2 != nil {
						log.Logger.Fatal(err2)
					}
				}
				log.Logger.Info("Graceful shutdown complete")
				os.Exit(1)
			}()
		},
	}

	cmd.AddCommand(
		newTemplateCmd(),
		newInstallCmd(),
		newDeleteCmd(),
		newCleanCmd(),
		newOutputCmd(),
		newVarsCmd(),
		newValidateCmd(),
		newGraphCmd(),
	)

	cmd.Aliases = []string{"kapp"}

	return cmd
}
