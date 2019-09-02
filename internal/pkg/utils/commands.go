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
	"io"
	"os"
	"os/exec"
	"sort"
	"strings"
	"time"
)

// Executes a command with an optional timeout, writing stdout and stderr to
// buffers. If `dryRun` is true, a log message of what would have been executed
// is emitted instead.
func ExecCommand(command string, args []string, envVars map[string]string,
	stdoutBuf *bytes.Buffer, stderrBuf *bytes.Buffer, dir string,
	timeoutSeconds int, expectedExitCode int, dryRun bool) error {

	// reset the buffers in case they've already been used
	stdoutBuf.Reset()
	stderrBuf.Reset()

	strEnvVars := make([]string, len(envVars))
	for k, v := range envVars {
		strEnvVars = append(strEnvVars, strings.Join([]string{k, v}, "="))
	}

	// sort the env vars to simplify copying and pasting log output
	sort.Strings(strEnvVars)

	log.Logger.Infof("Command '%s' has args: %#v and explicit env vars: %#v", command, args, strEnvVars)

	completeEnvVars := append(os.Environ(), strEnvVars...)
	sort.Strings(completeEnvVars)

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

		cmd = exec.CommandContext(ctx, command, args...)
	} else {
		cmd = exec.Command(command, args...)
	}

	cmd.Env = completeEnvVars
	cmd.Stdout = stdoutBuf
	cmd.Stderr = stderrBuf

	if dir != "" {
		cmd.Dir = dir
	}

	commandString := fmt.Sprintf("%s %s %s",
		strings.TrimSpace(strings.Join(strEnvVars, " ")),
		command, strings.Join(args, " "))

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
		if exitError, ok := err.(*exec.ExitError); ok {
			log.Logger.Infof("Command '%s' exited with a '%d', expected '%d'", commandString,
				exitError.ExitCode(), expectedExitCode)
			// if the command exited with an unexpected code, return a message
			if exitError.ExitCode() != expectedExitCode {
				return errors.Wrapf(err, "Failed to run command '%s' in directory '%s'. It exited with a code "+
					"of '%d' but '%d' was expected.\n"+
					"Stdout=%s\nStderr=%s", commandString, cmd.Dir, exitError.ExitCode(), expectedExitCode,
					stdoutBuf.String(), stderrBuf.String())
			}
		} else {
			return errors.Wrapf(err, "Failed to run command in directory '%s':\n%s\n"+
				"Stdout=%s\nStderr=%s", cmd.Dir, commandString, stdoutBuf.String(),
				stderrBuf.String())
		}
	} else {
		log.Logger.Infof("Command '%s' exited with a '%d', expected '%d'", commandString,
			cmd.ProcessState.ExitCode(), expectedExitCode)
		// if it exits cleanly but we expected a different code, that's still an error
		if cmd.ProcessState.ExitCode() != expectedExitCode {
			return fmt.Errorf("The command '%s' executed in '%s' exited with a code "+
				"of '%d', but '%d' was expected", commandString, cmd.Dir, cmd.ProcessState.ExitCode(),
				expectedExitCode)
		}
	}

	return nil
}

// Executes a command with an optional timeout, writing stdout and stderr to
// io.Writers directly. If `dryRun` is true, a log message of what would have been executed
// is emitted instead.
func ExecCommandUnbuffered(command string, args []string, envVars map[string]string,
	stdoutBuf io.Writer, stderrBuf io.Writer, dir string,
	timeoutSeconds int, expectedExitCode int, dryRun bool) error {

	strEnvVars := make([]string, len(envVars))
	for k, v := range envVars {
		strEnvVars = append(strEnvVars, strings.Join([]string{k, v}, "="))
	}

	// sort the env vars to simplify copying and pasting log output
	sort.Strings(strEnvVars)

	log.Logger.Infof("Command '%s' has args: %#v and explicit env vars: %#v", command, args, strEnvVars)

	completeEnvVars := append(os.Environ(), strEnvVars...)
	sort.Strings(completeEnvVars)

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

		cmd = exec.CommandContext(ctx, command, args...)
	} else {
		cmd = exec.Command(command, args...)
	}

	cmd.Env = completeEnvVars
	cmd.Stdout = stdoutBuf
	cmd.Stderr = stderrBuf

	if dir != "" {
		cmd.Dir = dir
	}

	commandString := fmt.Sprintf("%s %s %s",
		strings.TrimSpace(strings.Join(strEnvVars, " ")),
		command, strings.Join(args, " "))

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
			"directory '%s':\n%s\n", cmd.Dir, commandString)
	}
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			log.Logger.Infof("Command '%s' exited with a '%d', expected '%d'", commandString,
				exitError.ExitCode(), expectedExitCode)
			// if the command exited with an unexpected code, return a message
			if exitError.ExitCode() != expectedExitCode {
				return errors.Wrapf(err, "Failed to run command '%s' in directory '%s'. It exited with a code "+
					"of '%d' but '%d' was expected.\n", commandString, cmd.Dir, exitError.ExitCode(), expectedExitCode)
			}
		} else {
			return errors.Wrapf(err, "Failed to run command in directory '%s':\n%s\n",
				cmd.Dir, commandString)
		}
	} else {
		log.Logger.Infof("Command '%s' exited with a '%d', expected '%d'", commandString,
			cmd.ProcessState.ExitCode(), expectedExitCode)
		// if it exits cleanly but we expected a different code, that's still an error
		if cmd.ProcessState.ExitCode() != expectedExitCode {
			return fmt.Errorf("The command '%s' executed in '%s' exited with a code "+
				"of '%d', but '%d' was expected", commandString, cmd.Dir, cmd.ProcessState.ExitCode(),
				expectedExitCode)
		}
	}

	return nil
}
