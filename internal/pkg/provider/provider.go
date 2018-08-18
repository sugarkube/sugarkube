package provider

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/sugarkube/sugarkube/internal/pkg/log"
	"github.com/sugarkube/sugarkube/internal/pkg/vars"
	"path/filepath"
)

const valuesFile = "values.yaml"

type Provider interface {
	// Method that returns all paths in a config directory relevant to the
	// target profile/cluster/region, etc. that should be searched for values
	// files to merge.
	VarsDirs(sc *vars.StackConfig) []string
}

// Factory that creates providers
func NewProvider(name string) (Provider, error) {
	if name == "local" {
		return LocalProvider{}, nil
	}

	if name == "aws" {
		return AwsProvider{}, nil
	}

	return nil, errors.New(fmt.Sprintf("Provider '%s' doesn't exist", name))
}

// Searches for values.yaml files in configured directories and returns the
// result of merging them.
func StackConfigVars(p Provider, sc *vars.StackConfig) (map[string]interface{}, error) {
	stackConfigVars := map[string]interface{}{}

	for _, varFile := range p.VarsDirs(sc) {
		varDir := filepath.Join(sc.Dir(), varFile)
		groupedFiles, err := vars.GroupFiles(varDir)
		if err != nil {
			return nil, errors.WithStack(err)
		}

		valuesPaths, ok := groupedFiles[valuesFile]
		if !ok {
			log.Debugf("Skipping loading vars from directory %s. No %s file",
				varDir, valuesFile)
			continue
		}

		for _, path := range valuesPaths {
			err := vars.Merge(&stackConfigVars, path)
			if err != nil {
				return nil, errors.WithStack(err)
			}
		}
	}

	return stackConfigVars, nil
}
