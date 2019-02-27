package kapp

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestMergeVarsForKapp(t *testing.T) {

	// testing the correctness of this stack is handled in stack_test.go
	stackConfig, err := LoadStackConfig("large", "../../testdata/stacks.yaml")
	assert.Nil(t, err)
	assert.NotNil(t, stackConfig)

	expectedVarsFromFiles := map[string]interface{}{
		"colours": []interface{}{
			"green",
		},
	}

	results, err := stackConfig.GetKappVarsFromFiles(&stackConfig.Manifests[0].ParsedKapps()[0])
	assert.Nil(t, err)

	assert.Equal(t, expectedVarsFromFiles, results)

	// now we've loaded kapp variables from a file, test merging vars for the kapp

}
