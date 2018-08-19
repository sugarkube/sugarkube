package kapp

import (
	"github.com/sugarkube/sugarkube/internal/pkg/acquirer"
)

type installerConfig struct {
	kapp         string
	searchValues []string
	params       map[string]string
}

type Kapp struct {
	id              string
	installerConfig installerConfig
	sources         []acquirer.Acquirer
}
