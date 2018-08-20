// +build integration

package acquirer

import (
	"github.com/stretchr/testify/assert"
	"github.com/sugarkube/sugarkube/internal/pkg/log"
	"io/ioutil"
	"os"
	"testing"
)

func TestGitAcquire(t *testing.T) {
	acquirer, err := NewAcquirer(defaultSettings)
	assert.Nil(t, err)

	tempDir, err := ioutil.TempDir("", "git-")
	assert.Nil(t, err)

	log.Infof("Testing the git acquirer with temp dir: %s", tempDir)
	defer os.RemoveAll(tempDir)

	err = acquirer.Acquire(tempDir)
	assert.Nil(t, err)
}
