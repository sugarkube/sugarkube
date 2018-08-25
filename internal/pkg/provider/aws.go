package provider

import (
	"github.com/sugarkube/sugarkube/internal/pkg/kapp"
)

type AwsProvider struct {
	stackConfigVars Values
	region          string // todo - set this to the dir name when parsing variables
}

func (p *AwsProvider) varsDirs(stackConfig *kapp.StackConfig) ([]string, error) {
	return []string{
		"/cat",
		"/dog",
	}, nil
}

// Associate provider variables with the provider
func (p *AwsProvider) setVars(values Values) {
	p.stackConfigVars = values
}

// Returns the variables loaded by the Provider
func (p *AwsProvider) getVars() Values {
	return p.stackConfigVars
}

// Return vars loaded from configs that should be passed on to all kapps by
// installers so kapps can be installed into this provider
func (p *AwsProvider) getInstallerVars() Values {
	return Values{
		"REGION": p.region,
	}
}
