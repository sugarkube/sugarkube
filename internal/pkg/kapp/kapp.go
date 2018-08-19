package kapp

import (
	"github.com/pkg/errors"
	"github.com/sugarkube/sugarkube/internal/pkg/acquirer"
	"github.com/sugarkube/sugarkube/internal/pkg/log"
	"github.com/sugarkube/sugarkube/internal/pkg/vars"
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

// Parses a manifest file and returns a list of kapps on success
func parseManifest(manifest string) ([]Kapp, error) {
	log.Debugf("Parsing manifest: %s", manifest)

	kapps := make([]Kapp, 0)

	data, err := vars.LoadYamlFile(manifest)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	log.Debugf("Loaded manifest data: %#v", data)

	return kapps, nil
}

// Parses manifest files and returns a list of kapps on success
func ParseManifests(manifests []string) ([]Kapp, error) {
	log.Debugf("Parsing %d manifest(s)", len(manifests))

	kapps := make([]Kapp, 0)

	for _, manifest := range manifests {
		manifestKapps, err := parseManifest(manifest)
		if err != nil {
			return nil, errors.WithStack(err)
		}

		kapps = append(kapps, manifestKapps...)
	}

	return kapps, nil
}
