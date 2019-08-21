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

package utils

import (
	"bytes"
	"context"
	"fmt"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/sugarkube/sugarkube/internal/pkg/log"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strings"
	"time"
)

// Executes a command with an optional timeout, writing stdout and stderr to
// buffers. If `dryRun` is true, a log message of what would have been executed
// is emitted instead.
func ExecCommand(command string, args []string, envVars map[string]string,
	stdoutBuf *bytes.Buffer, stderrBuf *bytes.Buffer, dir string,
	timeoutSeconds int, dryRun bool) error {

	// reset the buffers in case they've already been used
	stdoutBuf.Reset()
	stderrBuf.Reset()

	strEnvVars := make([]string, len(envVars))
	for k, v := range envVars {
		strEnvVars = append(strEnvVars, strings.Join([]string{k, v}, "="))
	}

	// sort the env vars to simplify copying and pasting log output
	sort.Strings(strEnvVars)

	// flatten the args list if it contains sublists. Also remove empty lists
	sanitisedArgs := make([]string, 0)
	listRegEx := regexp.MustCompile(`\[([^]]+)]`)
	for _, arg := range args {
		log.Logger.Tracef("Searching command arg '%s' for sublists...", arg)
		reMatches := listRegEx.FindAllStringSubmatch(arg, -1)

		if len(reMatches) > 0 {
			for _, matches := range reMatches {
				log.Logger.Tracef("Command arg contains sublists: %#v", matches)
				match := strings.TrimSpace(matches[1])
				if match != "" {
					sanitisedArgs = append(sanitisedArgs, match)
				}
			}
		} else {
			// ignore empty arguments
			arg = strings.TrimSpace(arg)
			if arg != "" {
				// just append the argument
				sanitisedArgs = append(sanitisedArgs, arg)
			}
		}
	}

	log.Logger.Infof("Command '%s' has args: %#v and explicit env vars: %#v", command, sanitisedArgs, strEnvVars)

	completeEnvVars := append(os.Environ(), strEnvVars...)

	if log.Logger.Level == logrus.TraceLevel || log.Logger.Level == logrus.DebugLevel {
		log.Logger.Debugf("Complete env vars are: %#v", completeEnvVars)
	}

	var cmd *exec.Cmd
	var ctx context.Context
	var cancel context.CancelFunc

	if timeoutSeconds > 0 {
		log.Logger.Debugf("%s command will be run with a timeout of %d seconds",
			command, timeoutSeconds)

		ctx, cancel = context.WithTimeout(context.Background(),
			time.Duration(timeoutSeconds)*time.Second)
		defer cancel() // The cancel should be deferred so resources are cleaned up

		cmd = exec.CommandContext(ctx, command, sanitisedArgs...)
	} else {
		cmd = exec.Command(command, sanitisedArgs...)
	}

	cmd.Env = completeEnvVars
	cmd.Stdout = stdoutBuf
	cmd.Stderr = stderrBuf

	if dir != "" {
		cmd.Dir = dir
	}

	commandString := fmt.Sprintf("%s %s %s",
		strings.TrimSpace(strings.Join(strEnvVars, " ")),
		command, strings.Join(sanitisedArgs, " "))

	if dryRun {
		log.Logger.Infof("Dry run. Would run command in directory '%s':\n%s\n",
			cmd.Dir, commandString)
		return nil
	} else {
		log.Logger.Infof("Executing command in directory '%s':\n%s\n",
			cmd.Dir, commandString)
	}

	err := cmd.Run()
	if timeoutSeconds > 0 && ctx.Err() == context.DeadlineExceeded {
		return errors.Wrapf(ctx.Err(), "Timed out executing command in "+
			"directory '%s':\n%s\nStdout=%s\nStderr=%s", cmd.Dir, commandString,
			stdoutBuf.String(), stderrBuf.String())
	}
	if err != nil {
		return errors.Wrapf(err, "Failed to run command in directory '%s':\n%s\n"+
			"Stdout=%s\nStderr=%s", cmd.Dir, commandString, stdoutBuf.String(),
			stderrBuf.String())
	}

	return nil
}
