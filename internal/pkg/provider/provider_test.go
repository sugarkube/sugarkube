package provider

import (
	"github.com/stretchr/testify/assert"
	"github.com/sugarkube/sugarkube/internal/pkg/vars"
	"testing"
)

func TestStackConfigVars(t *testing.T) {
	sc, err := vars.LoadStackConfig("large", "../vars/testdata/stacks.yaml")
	assert.Nil(t, err)

	expected := &Values{
		"provisioner": map[interface{}]interface{}{
			"memory":    4096,
			"cpus":      4,
			"disk_size": "120g",
		},
	}

	providerImpl, err := NewProvider(sc.Provider)
	assert.Nil(t, err)

	actual, err := StackConfigVars(providerImpl, sc)
	assert.Nil(t, err)
	assert.Equal(t, expected, actual, "Mismatching vars")
}
