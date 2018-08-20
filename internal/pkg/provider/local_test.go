package provider

import (
	"github.com/stretchr/testify/assert"
	"github.com/sugarkube/sugarkube/internal/pkg/kapp"
	"testing"
)

func TestLocalVarsDirs(t *testing.T) {
	sc, err := kapp.LoadStackConfig("large", "../../testdata/stacks.yaml")
	assert.Nil(t, err)

	expected := []string{
		"../../testdata/stacks",
		"../../testdata/stacks/local",
		"../../testdata/stacks/local/profiles",
		"../../testdata/stacks/local/profiles/local",
		"../../testdata/stacks/local/profiles/local/clusters",
		"../../testdata/stacks/local/profiles/local/clusters/large",
	}

	provider := LocalProvider{}
	actual, err := provider.VarsDirs(sc)
	assert.Nil(t, err)

	assert.Equal(t, expected, actual, "Incorrect vars dirs returned")
}
