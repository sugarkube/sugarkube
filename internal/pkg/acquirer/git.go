package acquirer

type GitAcquirer struct {
	Acquirer
}

// Acquires kapps via git.
func (a GitAcquirer) Acquire(path string) error {
	return nil
}
