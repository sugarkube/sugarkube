package provider

import (
	"github.com/stretchr/testify/assert"
	"github.com/sugarkube/sugarkube/internal/pkg/vars"
	"testing"
)

func TestLocalVarsDirs(t *testing.T) {
	sc := vars.StackConfig{
		Profile: "test-profile",
		Cluster: "test-cluster",
		VarsFilesDirs: []string{
			"./testdata",
		},
	}

	expected := []string{
		"testdata",
		"testdata/profiles",
		"testdata/profiles/test-profile",
		"testdata/profiles/test-profile/clusters",
		"testdata/profiles/test-profile/clusters/test-cluster",
	}

	provider := LocalProvider{}
	actual := provider.VarsDirs(&sc)

	assert.Equal(t, expected, actual, "Incorrect vars dirs returned")
}
