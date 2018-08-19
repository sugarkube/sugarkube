package acquirer

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/sugarkube/sugarkube/internal/pkg/log"
	"strings"
)

type Acquirer interface {
	Acquire(path string) error
}

const GIT = "git"

// Factory that creates acquirers
func newAcquirer(name string) (Acquirer, error) {
	log.Debugf("Returning new %s acquirer", name)

	if name == GIT {
		return GitAcquirer{}, nil
	}

	return nil, errors.New(fmt.Sprintf("Acquirer '%s' doesn't exist", name))
}

// Identifies the requirer for a given path, and returns a new instance of it
func NewAcquirerForPath(path string) (Acquirer, error) {
	if strings.HasPrefix(path, "git") {
		return newAcquirer(GIT)
	}

	return nil, errors.New(fmt.Sprintf("Couldn't identify acquirer for path '%s'", path))
}
