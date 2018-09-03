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

package acquirer

import (
	"bytes"
	"fmt"
	"github.com/pkg/errors"
	"github.com/sugarkube/sugarkube/internal/pkg/log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type GitAcquirer struct {
	name   string
	uri    string
	branch string
	path   string
}

// todo - make configurable
const GIT_PATH = "git"

const NAME = "name"
const URI = "uri"
const BRANCH = "branch"
const PATH = "path"

// Returns an instance. This allows us to build objects for testing instead of
// directly instantiating objects in the acquirer factory.
func NewGitAcquirer(name string, uri string, branch string, path string) GitAcquirer {
	if name == "" {
		name = filepath.Base(path)
	}

	return GitAcquirer{
		name:   name,
		uri:    uri,
		branch: branch,
		path:   path,
	}
}

// Generate an ID
func (a GitAcquirer) Id() (string, error) {
	// testing here simplifies testing but does mean invalid objects can be created...
	if strings.Count(a.uri, ":") != 1 {
		return "", errors.New(
			fmt.Sprintf("Unexpected git URI. Expected a single ':' "+
				"character in URI %s", a.uri))
	}

	orgRepo := strings.SplitAfter(a.uri, ":")
	hyphenatedOrg := strings.Replace(orgRepo[1], "/", "-", -1)
	hyphenatedOrg = strings.TrimSuffix(hyphenatedOrg, ".git")
	hyphenatedBranc := strings.Replace(a.branch, "/", "-", -1)
	hyphenatedName := strings.Replace(a.name, "/", "-", -1)

	return strings.Join([]string{hyphenatedOrg, hyphenatedBranc, hyphenatedName}, "-"), nil
}

// return the name
func (a GitAcquirer) Name() string {
	return a.name
}

// return the path
func (a GitAcquirer) Path() string {
	return a.path
}

// Acquires kapps via git and saves them to `dest`.
func (a GitAcquirer) acquire(dest string) error {

	log.Infof("Acquiring git source %s into %s", a.uri, dest)

	// create the dest dir if it doesn't exist
	err := os.MkdirAll(dest, 0755)
	if err != nil {
		return errors.Wrapf(err, "Error creating directory %s", dest)
	}

	userEnv := os.Environ()

	var stderrBuf bytes.Buffer

	// git init
	initCmd := exec.Command(GIT_PATH, "init")
	initCmd.Dir = dest
	initCmd.Env = userEnv
	initCmd.Stderr = &stderrBuf
	err = initCmd.Run()
	if err != nil {
		return errors.Wrapf(err, "Error running: %s. Stderr: %s",
			strings.Join(initCmd.Args, " "), stderrBuf.String())
	}

	stderrBuf.Reset()

	// add origin
	remoteAddCmd := exec.Command(GIT_PATH, "remote", "add", "origin", a.uri)
	remoteAddCmd.Dir = dest
	remoteAddCmd.Env = userEnv
	remoteAddCmd.Stderr = &stderrBuf
	err = remoteAddCmd.Run()
	if err != nil {
		return errors.Wrapf(err, "Error running: %s. Stderr=%s",
			strings.Join(remoteAddCmd.Args, " "), stderrBuf.String())
	}

	stderrBuf.Reset()

	fetchCmd := exec.Command(GIT_PATH, "fetch")
	fetchCmd.Dir = dest
	fetchCmd.Env = userEnv
	fetchCmd.Stderr = &stderrBuf
	err = fetchCmd.Run()
	if err != nil {
		return errors.Wrapf(err, "Error running: %s. Stderr=%s",
			strings.Join(fetchCmd.Args, " "), stderrBuf.String())
	}

	stderrBuf.Reset()

	configCmd := exec.Command(GIT_PATH, "config", "core.sparsecheckout", "true")
	configCmd.Dir = dest
	configCmd.Env = userEnv
	configCmd.Stderr = &stderrBuf
	err = configCmd.Run()
	if err != nil {
		return errors.Wrapf(err, "Error running: %s. Stderr=%s",
			strings.Join(configCmd.Args, " "), stderrBuf.String())
	}

	err = appendToFile(filepath.Join(dest, ".git/info/sparse-checkout"),
		fmt.Sprintf("%s/*\n", strings.TrimSuffix(a.path, "/")))
	if err != nil {
		return errors.WithStack(err)
	}

	stderrBuf.Reset()

	checkoutCmd := exec.Command(GIT_PATH, "checkout", a.branch)
	checkoutCmd.Dir = dest
	checkoutCmd.Env = userEnv
	checkoutCmd.Stderr = &stderrBuf
	err = checkoutCmd.Run()
	if err != nil {
		return errors.Wrapf(err, "Error running: '%s' on %s with path '%s'. "+
			"Stderr=%s", strings.Join(
			checkoutCmd.Args, " "), a.uri, a.path, stderrBuf.String())
	}

	// we could optionally verify tags with:
	// git tag -v a.branch 2>&1 >/dev/null | grep -E '{{ trusted_gpg_keys|join('|') }}'

	return nil
}

// Appends text to a file
func appendToFile(filename string, text string) error {
	// create the file if it doesn't exist
	f, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0744)
	if err != nil {
		return errors.Wrapf(err, "Error opening file %s", filename)
	}

	defer f.Close()

	if _, err = f.WriteString(text); err != nil {
		return errors.Wrapf(err, "Error writing to file %s", filename)
	}

	return nil
}
