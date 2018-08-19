package acquirer

type GitAcquirer struct {
	Acquirer
	url    string
	branch string
	path   string
}

// Acquires kapps via git.
func (a GitAcquirer) Acquire(path string) error {
	return nil
}
