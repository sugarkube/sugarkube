package sot

type Sot interface {
	refresh() error
	isInstalled(name string, version string) (bool, error)
}
