package provider

import "github.com/sugarkube/sugarkube/internal/pkg/vars"

type AwsProvider struct {
	Provider
}

func (p AwsProvider) VarsDirs(stackConfig *vars.StackConfig) []string {
	return []string{
		"/cat",
		"/dog",
	}
}
