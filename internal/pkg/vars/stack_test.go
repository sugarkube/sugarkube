package vars

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestLoadStackConfigGarbagePath(t *testing.T) {
	_, err := LoadStackConfig("fake-path", "/fake/~/some?/~/garbage")
	assert.Error(t, err)
}

func TestLoadStackConfigNonExistentPath(t *testing.T) {
	_, err := LoadStackConfig("missing-path", "/missing/stacks.yaml")
	assert.Error(t, err)
}

func TestLoadStackConfigDir(t *testing.T) {
	_, err := LoadStackConfig("dir-path", "./testdata")
	assert.Error(t, err)
}

func TestLoadStackConfig(t *testing.T) {
	expected := &StackConfig{
		Name:        "local-large-test",
		Provider:    "local",
		Provisioner: "minikube",
		Profile:     "local",
		Cluster:     "large",
		VarsFilesDirs: []string{
			"providers/minikube/",
		},
		Manifests: []string{
			"./testdata/manifest1.yaml",
			"./testdata/manifest2.yaml",
		},
	}

	actual, err := LoadStackConfig("local-large-test", "./testdata/stacks.yaml")
	assert.Nil(t, err)
	assert.Equal(t, expected, actual, "unexpected stack")
}

func TestLoadStackConfigMissingStackName(t *testing.T) {
	_, err := LoadStackConfig("missing-stack-name", "./testdata/stacks.yaml")
	assert.Error(t, err)
}

func TestStackConfigVars(t *testing.T) {
	stack, err := LoadStackConfig("local-large-test", "./testdata/stacks.yaml")
	assert.Nil(t, err)

	expected := map[string]interface{}{
		"provisioner_params": map[string]interface{}{
			"memory":    4096,
			"cpus":      4,
			"disk_size": "120g",
		},
	}

	actual := stack.Vars()
	assert.Equal(t, expected, actual, "Mismatching vars")
}
