package provider

import "github.com/sugarkube/sugarkube/internal/pkg/vars"

type LocalProvider struct {
	Provider
}

func (p LocalProvider) VarsDirs(stackConfig *vars.StackConfig) []string {

	return []string{
		"/cat",
		"/dog",
	}
}
