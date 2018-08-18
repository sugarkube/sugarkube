package provisioner

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestNewMinikubeProvisioner(t *testing.T) {
	actual, err := NewProvisioner(MINIKUBE)
	assert.Nil(t, err)
	assert.Equal(t, MinikubeProvisioner{}, actual)
}

func TestNewKOPSProvisioner(t *testing.T) {
	actual, err := NewProvisioner(KOPS)
	assert.Nil(t, err)
	assert.Equal(t, KopsProvisioner{}, actual)
}
