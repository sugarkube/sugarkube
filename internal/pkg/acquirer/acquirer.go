package acquirer

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/sugarkube/sugarkube/internal/pkg/log"
	"strings"
)

type Acquirer interface {
	Acquire(dest string) error
}

const ACQUIRER_KEY = "acquirer"
const GIT = "git"

// Factory that creates acquirers
func newAcquirer(name string, settings map[string]string) (Acquirer, error) {
	log.Debugf("Returning new %s acquirer", name)

	if name == GIT {
		return GitAcquirer{
			url:    settings["url"],
			branch: settings["branch"],
			path:   settings["path"],
		}, nil
	}

	return nil, errors.New(fmt.Sprintf("Acquirer '%s' doesn't exist", name))
}

// Identifies the requirer for a given path, and returns a new instance of it
func NewAcquirerForPath(path string, settings map[string]string) (Acquirer, error) {
	// perhaps the acquirer is explicitly declared in settings
	acquirer := settings[ACQUIRER_KEY]

	if strings.HasPrefix(path, GIT) || acquirer == GIT {
		return newAcquirer(GIT, settings)
	}

	return nil, errors.New(fmt.Sprintf("Couldn't identify acquirer for path '%s'", path))
}
