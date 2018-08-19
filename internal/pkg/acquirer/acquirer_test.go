package acquirer

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestNewAcquirerError(t *testing.T) {
	actual, err := newAcquirer("nonsense")
	assert.NotNil(t, err)
	assert.Nil(t, actual)
}

func TestNewGitAcquirer(t *testing.T) {
	actual, err := newAcquirer(GIT)
	assert.Nil(t, err)
	assert.Equal(t, GitAcquirer{}, actual)
}

func TestNewAcquirerForPathGit(t *testing.T) {
	actual, err := NewAcquirerForPath("git@github.com:sugarkube/sugarkube.git")
	assert.Nil(t, err)
	assert.Equal(t, GitAcquirer{}, actual)
}

func TestNewAcquirerForPathError(t *testing.T) {
	actual, err := NewAcquirerForPath("nonsense")
	assert.NotNil(t, err)
	assert.Nil(t, actual)
}
