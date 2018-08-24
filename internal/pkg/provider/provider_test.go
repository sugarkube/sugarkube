package provider

import (
	"github.com/stretchr/testify/assert"
	"github.com/sugarkube/sugarkube/internal/pkg/kapp"
	"testing"
)

func TestStackConfigVars(t *testing.T) {
	sc, err := kapp.LoadStackConfig("large", "../../testdata/stacks.yaml")
	assert.Nil(t, err)

	expected := Values{
		"provisioner": map[interface{}]interface{}{
			"memory":    4096,
			"cpus":      4,
			"disk_size": "120g",
		},
	}

	providerImpl, err := newProviderImpl(sc.Provider)
	assert.Nil(t, err)

	actual, err := stackConfigVars(providerImpl, sc)
	assert.Nil(t, err)
	assert.Equal(t, expected, actual, "Mismatching vars")
}

func TestNewProviderError(t *testing.T) {
	actual, err := newProviderImpl("nonsense")
	assert.NotNil(t, err)
	assert.Nil(t, actual)
}

func TestNewLocalProvider(t *testing.T) {
	actual, err := newProviderImpl(LOCAL)
	assert.Nil(t, err)
	assert.Equal(t, &LocalProvider{}, actual)
}

func TestNewAWSProvider(t *testing.T) {
	actual, err := newProviderImpl(AWS)
	assert.Nil(t, err)
	assert.Equal(t, &AwsProvider{}, actual)
}
