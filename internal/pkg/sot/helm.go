package sot

// Uses Helm to determine which kapps are already installed in a target cluster
type HelmSot struct{}

func (s HelmSot) Refresh() error {
	panic("not implemented")
}

func (s HelmSot) IsInstalled(name string, version string) (bool, error) {
	panic("not implemented")
}
