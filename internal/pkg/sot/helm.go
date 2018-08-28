package sot

import (
	"bytes"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
	"os"
	"os/exec"
)

// Uses Helm to determine which kapps are already installed in a target cluster
type HelmSot struct{}

// Wrapper around Helm output
type HelmOutput struct {
	Next     string
	Releases []HelmRelease
}

// struct returned by `helm list --output yaml`
type HelmRelease struct {
	AppVersion string
	Chart      string
	Name       string
	Namespace  string
	Revision   int
	Status     string
	Updated    string
}

// Refresh the list of Helm charts
func (s HelmSot) refresh() error {
	var stdout bytes.Buffer
	cmd := exec.Command("helm", "list", "--output", "yaml")
	cmd.Env = os.Environ()
	cmd.Stdout = &stdout

	err := cmd.Run()
	if err != nil {
		return errors.Wrap(err, "Error running 'helm list'")
	}

	// parse stdout
	output := HelmOutput{}
	err = yaml.Unmarshal(stdout.Bytes(), &output)
	if err != nil {
		return errors.Wrapf(err, "Error parsing 'Helm list' output: %s",
			stdout.String())
	}

	return nil
}

func (s HelmSot) isInstalled(name string, version string) (bool, error) {
	panic("not implemented")
}
