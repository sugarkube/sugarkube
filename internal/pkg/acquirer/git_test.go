package acquirer

import (
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"testing"
)

func TestGitAcquire(t *testing.T) {
	acquirer, err := NewAcquirerForPath("git@github.com:sugarkube/sugarkube.git",
		defaultSettings)
	assert.Nil(t, err)

	tempDir, err := ioutil.TempDir("", "git-")
	assert.Nil(t, err)

	err = acquirer.Acquire(tempDir)
	assert.Nil(t, err)
}
