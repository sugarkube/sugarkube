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
	assert.NotNil(t, err)
	assert.Nil(t, actual)
}

var defaultSettings = map[string]string{
	"url":    "git@github.com:sugarkube/sugarkube.git",
	"branch": "tiller-0.1.0",
	"path":   "tiller/",
}

var expectedAcquirer = GitAcquirer{
	url:    "git@github.com:sugarkube/sugarkube.git",
	branch: "tiller-0.1.0",
	path:   "tiller/",
}

func TestNewGitAcquirerFull(t *testing.T) {
	actual, err := newAcquirer(GIT, defaultSettings)
	assert.Nil(t, err)
	assert.Equal(t, expectedAcquirer, actual,
		"Fully-defined git acquirer incorrectly created")
}

func TestNewAcquirerForPathGit(t *testing.T) {
	actual, err := NewAcquirerForPath("git@github.com:sugarkube/sugarkube.git",
		defaultSettings)
	assert.Nil(t, err)
	assert.Equal(t, expectedAcquirer, actual)
}

func TestNewAcquirerForPathGitExplicit(t *testing.T) {
	actual, err := NewAcquirerForPath("https://github.com/sugarkube/sugarkube.git",
		map[string]string{
			ACQUIRER_KEY: GIT,
			"url":        "git@github.com:sugarkube/sugarkube.git",
			"branch":     "tiller-0.1.0",
			"path":       "tiller/",
		})
	assert.Nil(t, err)
	assert.Equal(t, expectedAcquirer, actual)
}

func TestNewAcquirerForPathError(t *testing.T) {
	actual, err := NewAcquirerForPath("nonsense", map[string]string{})
	assert.NotNil(t, err)
	assert.Nil(t, actual)
}
