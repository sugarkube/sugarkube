package vars

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestLoadStackGarbagePath(t *testing.T) {
	_, err := LoadStack("fake-path", "/fake/~/some?/~/garbage")
	assert.Error(t, err)
}

func TestLoadStackNonExistentPath(t *testing.T) {
	_, err := LoadStack("missing-path", "/missing/stacks.yaml")
	assert.Error(t, err)
}

func TestLoadStackDir(t *testing.T) {
	_, err := LoadStack("dir-path", "./testdata")
	assert.Error(t, err)
}

func TestLoadStack(t *testing.T) {
	expected := &Stack{
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

	actual, err := LoadStack("local-large-test", "./testdata/stacks.yaml")
	assert.Nil(t, err)
	assert.Equal(t, expected, actual, "unexpected stack")
}

func TestLoadStackMissingStackName(t *testing.T) {
	_, err := LoadStack("missing-stack-name", "./testdata/stacks.yaml")
	assert.Error(t, err)
}
