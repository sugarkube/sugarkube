package acquirer

type GitAcquirer struct {
	Acquirer
	url    string
	branch string
	path   string
}

// todo - make configurable
const GIT_PATH = "git"

// Acquires kapps via git and saves them to `dest`.
func (a GitAcquirer) Acquire(dest string) error {

	return nil
}
