package kappsot

type KappSot interface {
	refresh() error
	isInstalled(name string, version string) (bool, error)
}
