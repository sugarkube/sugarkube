package provider

import (
	"github.com/stretchr/testify/assert"
	"github.com/sugarkube/sugarkube/internal/pkg/vars"
	"testing"
)

func TestLocalVarsDirs(t *testing.T) {
	sc, err := vars.LoadStackConfig("large", "../vars/testdata/stacks.yaml")
	assert.Nil(t, err)

	expected := []string{
		"../vars/testdata/stacks",
		"../vars/testdata/stacks/local",
		"../vars/testdata/stacks/local/profiles",
		"../vars/testdata/stacks/local/profiles/local",
		"../vars/testdata/stacks/local/profiles/local/clusters",
		"../vars/testdata/stacks/local/profiles/local/clusters/large",
	}

	provider := LocalProvider{}
	actual, err := provider.VarsDirs(sc)
	assert.Nil(t, err)

	assert.Equal(t, expected, actual, "Incorrect vars dirs returned")
}
