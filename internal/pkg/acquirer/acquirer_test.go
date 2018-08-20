package acquirer

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestNewAcquirerError(t *testing.T) {
	actual, err := acquirerFactory("nonsense", map[string]string{})
	assert.NotNil(t, err)
	assert.Nil(t, actual)
}

func TestNewGitAcquirerPartial(t *testing.T) {
	actual, err := acquirerFactory(GIT, map[string]string{
		"branch": "master",
	})
	assert.NotNil(t, err)
	assert.Nil(t, actual)
}

var defaultSettings = map[string]string{
	"uri":    "git@github.com:sugarkube/sample-kapps.git",
	"branch": "tiller-0.1.0",
	"path":   "tiller/",
}

var expectedAcquirer = GitAcquirer{
	name:   "tiller",
	uri:    "git@github.com:sugarkube/sample-kapps.git",
	branch: "tiller-0.1.0",
	path:   "tiller/",
}

func TestNewGitAcquirerFull(t *testing.T) {
	actual, err := acquirerFactory(GIT, defaultSettings)
	assert.Nil(t, err)
	assert.Equal(t, expectedAcquirer, actual,
		"Fully-defined git acquirer incorrectly created")
}

func TestNewAcquirerGit(t *testing.T) {
	actual, err := NewAcquirer(defaultSettings)
	assert.Nil(t, err)
	assert.Equal(t, expectedAcquirer, actual)
}

func TestNewAcquirerGitExplicit(t *testing.T) {
	actual, err := NewAcquirer(
		map[string]string{
			ACQUIRER_KEY: GIT,
			"uri":        "git@github.com:sugarkube/sample-kapps.git",
			"branch":     "tiller-0.1.0",
			"path":       "tiller/",
		})
	assert.Nil(t, err)
	assert.Equal(t, expectedAcquirer, actual)
}

func TestNewAcquirerNilUriError(t *testing.T) {
	actual, err := NewAcquirer(map[string]string{
		"uri": "",
	})
	assert.NotNil(t, err)
	assert.Nil(t, actual)
}
