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
		"location": "kappFile",
	}

	kappObj := &stackConfig.Manifests[0].ParsedKapps()[0]

	results, err := stackConfig.GetVarsFromFiles(kappObj)
	assert.Nil(t, err)

	assert.Equal(t, expectedVarsFromFiles, results)

	// now we've loaded kapp variables from a file, test merging vars for the kapp
	expectedMergedVars := map[string]interface{}{
		"stack": map[interface{}]interface{}{
			"name":        "large",
			"profile":     "local",
			"provider":    "local",
			"provisioner": "minikube",
			"region":      "",
			"account":     "",
			"cluster":     "large",
			"filePath":    "../../testdata/stacks.yaml",
		},
		"sugarkube": map[interface{}]interface{}{
			"target":   "myTarget",
			"approved": true,
			"defaultVars": []interface{}{
				"local",
				"",
				"local",
				"large",
				"",
			},
		},
		"kapp": map[interface{}]interface{}{
			"id":        "kappA",
			"state":     "absent",
			"cacheRoot": "manifest1/kappA",
			"vars": map[interface{}]interface{}{
				"colours": []interface{}{
					"red",
					"black",
				},
				"location": "kappFile",
				"sizeVar":  "mediumOverridden",
				"stackVar": "setInOverrides",
			},
		},
	}

	mergedKappVars, err := MergeVarsForKapp(kappObj, stackConfig,
		map[string]interface{}{"target": "myTarget", "approved": true})

	assert.Equal(t, expectedMergedVars, mergedKappVars)
}
