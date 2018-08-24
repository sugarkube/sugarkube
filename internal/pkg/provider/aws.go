package provider

import (
	"github.com/sugarkube/sugarkube/internal/pkg/kapp"
)

type AwsProvider struct {
	stackConfigVars Values
	region          string // todo - set this to the dir name when parsing variables
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
func (p AwsProvider) getVars() Values {
	return p.stackConfigVars
}

// Return vars loaded from configs that should be passed on to kapps by Installers
func (p AwsProvider) getInstallerVars() Values {
	return Values{
		"KUBE_CONTEXT": p.stackConfigVars[KUBE_CONTEXT_KEY],
		"REGION":       p.region,
	}
}
