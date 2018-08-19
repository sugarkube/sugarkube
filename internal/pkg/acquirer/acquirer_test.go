package acquirer

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestNewAcquirerError(t *testing.T) {
	actual, err := newAcquirer("nonsense", map[string]string{})
	assert.NotNil(t, err)
	assert.Nil(t, actual)
}

func TestNewGitAcquirerPartial(t *testing.T) {
	actual, err := newAcquirer(GIT, map[string]string{
		"branch": "master",
	})
	assert.Nil(t, err)
	assert.Equal(t, GitAcquirer{branch: "master"}, actual,
		"Partial git acquirer incorrectly created")
}

func TestNewGitAcquirerFull(t *testing.T) {
	actual, err := newAcquirer(GIT, map[string]string{
		"url":    "git@github.com:sugarkube/sugarkube.git",
		"branch": "tiller-0.1.0",
		"path":   "tiller/",
	})
	assert.Nil(t, err)
	assert.Equal(t, GitAcquirer{
		url:    "git@github.com:sugarkube/sugarkube.git",
		branch: "tiller-0.1.0",
		path:   "tiller/",
	}, actual,
		"Fully-defined git acquirer incorrectly created")
}

func TestNewAcquirerForPathGit(t *testing.T) {
	actual, err := NewAcquirerForPath("git@github.com:sugarkube/sugarkube.git",
		map[string]string{})
	assert.Nil(t, err)
	assert.Equal(t, GitAcquirer{}, actual)
}

func TestNewAcquirerForPathError(t *testing.T) {
	actual, err := NewAcquirerForPath("nonsense", map[string]string{})
	assert.NotNil(t, err)
	assert.Nil(t, actual)
}
