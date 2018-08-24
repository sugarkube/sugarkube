package provider

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/sugarkube/sugarkube/internal/pkg/kapp"
	"github.com/sugarkube/sugarkube/internal/pkg/log"
	"github.com/sugarkube/sugarkube/internal/pkg/vars"
	"os"
	"path/filepath"
)

const valuesFile = "values.yaml"

type Values = map[string]interface{}

type Provider interface {
	// Method that returns all paths in a config directory relevant to the
	// target profile/cluster/region, etc. that should be searched for values
	// files to merge.
	varsDirs(sc *kapp.StackConfig) ([]string, error)
}

// implemented providers
const LOCAL = "local"
const AWS = "aws"

// Factory that creates providers
func NewProvider(name string) (Provider, error) {
	if name == LOCAL {
		return LocalProvider{}, nil
	}

	if name == AWS {
		return AwsProvider{}, nil
	}

	return nil, errors.New(fmt.Sprintf("Provider '%s' doesn't exist", name))
}

// Searches for values.yaml files in configured directories and returns the
// result of merging them.
func StackConfigVars(p Provider, sc *kapp.StackConfig) (Values, error) {
	stackConfigVars := Values{}

	varsDirs, err := p.varsDirs(sc)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	for _, varFile := range varsDirs {
		valuePath := filepath.Join(varFile, valuesFile)

		_, err := os.Stat(valuePath)
		if err != nil {
			log.Debugf("Skipping merging non-existent path %s", valuePath)
			continue
		}

		err = vars.Merge(&stackConfigVars, valuePath)
		if err != nil {
			return nil, errors.WithStack(err)
		}
	}

	return stackConfigVars, nil
}
