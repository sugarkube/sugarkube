package provider

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/sugarkube/sugarkube/internal/pkg/vars"
)

type Provider interface {
	VarsDirs(stackConfig *vars.StackConfig) []string
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
