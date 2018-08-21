package sot

type Sot interface {
	Refresh() error
	IsInstalled(name string, version string) (bool, error)
}
