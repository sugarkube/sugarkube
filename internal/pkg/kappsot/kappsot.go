package kappsot

type KappSot interface {
	refresh() error
	isInstalled(name string, version string) (bool, error)
}

// Delegate to an implementation
func IsInstalled(k KappSot, name string, version string) (bool, error) {
	return k.isInstalled(name, version)
}
