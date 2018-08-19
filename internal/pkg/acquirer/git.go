package acquirer

import (
	"fmt"
	"github.com/pkg/errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type GitAcquirer struct {
	Acquirer
	url    string
	branch string
	path   string
}

// todo - make configurable
const GIT_PATH = "git"

const URL = "url"
const BRANCH = "branch"
const PATH = "path"

// Acquires kapps via git and saves them to `dest`.
func (a GitAcquirer) Acquire(dest string) error {

	// create the dest dir if it doesn't exist
	err := os.MkdirAll(dest, 0755)
	if err != nil {
		return errors.Wrapf(err, "Error creating directory %s", dest)
	}

	// git init
	initCmd := exec.Command(GIT_PATH, "init")
	initCmd.Dir = dest
	err = initCmd.Run()
	if err != nil {
		return errors.Wrapf(err, "Error running: %s",
			strings.Join(initCmd.Args, " "))
	}

	// add origin
	remoteAddCmd := exec.Command(GIT_PATH, "remote", "add", "origin", a.url)
	remoteAddCmd.Dir = dest
	err = remoteAddCmd.Run()
	if err != nil {
		return errors.Wrapf(err, "Error running: %s",
			strings.Join(initCmd.Args, " "))
	}

	fetchCmd := exec.Command(GIT_PATH, "fetch")
	fetchCmd.Dir = dest
	err = fetchCmd.Run()
	if err != nil {
		return errors.Wrapf(err, "Error running: %s",
			strings.Join(initCmd.Args, " "))
	}

	configCmd := exec.Command(GIT_PATH, "config", "core.sparsecheckout", "true")
	configCmd.Dir = dest
	err = configCmd.Run()
	if err != nil {
		return errors.Wrapf(err, "Error running: %s",
			strings.Join(initCmd.Args, " "))
	}

	err = appendToFile(filepath.Join(dest, ".git/info/sparse-checkout"),
		fmt.Sprintf("%s/*\n", a.path))
	if err != nil {
		return errors.WithStack(err)
	}

	checkoutCmd := exec.Command(GIT_PATH, "checkout")
	checkoutCmd.Dir = dest
	err = checkoutCmd.Run()
	if err != nil {
		return errors.Wrapf(err, "Error running: %s",
			strings.Join(initCmd.Args, " "))
	}

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
