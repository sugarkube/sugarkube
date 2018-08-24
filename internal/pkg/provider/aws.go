package provider

import (
	"github.com/sugarkube/sugarkube/internal/pkg/kapp"
)

type AwsProvider struct{}

func (p AwsProvider) varsDirs(stackConfig *kapp.StackConfig) ([]string, error) {
	return []string{
		"/cat",
		"/dog",
	}, nil
}
