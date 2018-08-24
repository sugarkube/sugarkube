package provider

import (
	"github.com/sugarkube/sugarkube/internal/pkg/kapp"
)

type AwsProvider struct {
	stackConfigVars Values
}

func (p AwsProvider) varsDirs(stackConfig *kapp.StackConfig) ([]string, error) {
	return []string{
		"/cat",
		"/dog",
	}, nil
}

// Associate provider variables with the provider
func (p AwsProvider) setVars(values Values) {
	p.stackConfigVars = values
}

// Returns the variables loaded by the Provider
func (p AwsProvider) GetVars() Values {
	return p.stackConfigVars
}
