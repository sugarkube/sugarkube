package provider

import "github.com/sugarkube/sugarkube/internal/pkg/vars"

type AwsProvider struct{}

func (p AwsProvider) VarsDirs(stackConfig *vars.StackConfig) ([]string, error) {
	return []string{
		"/cat",
		"/dog",
	}, nil
}
