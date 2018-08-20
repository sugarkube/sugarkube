package kapp

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/sugarkube/sugarkube/internal/pkg/log"
	"github.com/sugarkube/sugarkube/internal/pkg/vars"
	"path/filepath"
	"strings"
)

type Manifest struct {
	// defaults to the file basename, but can be explicitly specified to avoid
	// clashes. This is also used to namespace entries in the cache
	Id    string
	Path  string
	Kapps []Kapp
}

func NewManifest(path string) Manifest {
	manifest := Manifest{
		Path: path,
	}

	SetManifestDefaults(&manifest)
	return manifest
}

// Sets fields to default values
func SetManifestDefaults(manifest *Manifest) {
	// use the basename after stripping the extension by default
	defaultId := strings.Replace(filepath.Base(manifest.Path), filepath.Ext(manifest.Path), "", 1)

	if manifest.Id == "" {
		manifest.Id = defaultId
	}
}

// Load a single manifest file and parse the kapps it defines
func parseManifestFile(path string) (*Manifest, error) {
	log.Debugf("Parsing manifest: %s", path)

	data, err := vars.LoadYamlFile(path)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	log.Debugf("Loaded manifest data: %#v", data)

	kapps, err := parseManifestYaml(data)

	manifest := Manifest{
		Id:    filepath.Base(path),
		Path:  path,
		Kapps: kapps,
	}

	return &manifest, nil
}

// Parses manifest files and returns a list of kapps on success
func ParseManifests(manifestPaths []string) ([]Manifest, error) {
	log.Debugf("Parsing %d manifest(s)", len(manifestPaths))

	manifests := make([]Manifest, 0)

	for _, manifestPath := range manifestPaths {
		manifest, err := parseManifestFile(manifestPath)
		if err != nil {
			return nil, errors.WithStack(err)
		}

		manifests = append(manifests, *manifest)
	}

	return manifests, nil
}

// Validates that there aren't multiple kapps with the same ID in the manifest,
// or it'll break creating a cache
func ValidateManifest(manifest *Manifest) error {
	ids := map[string]bool{}

	for _, kapp := range manifest.Kapps {
		id := kapp.id

		if _, ok := ids[id]; ok {
			return errors.New(fmt.Sprintf("Multiple kapps exist with "+
				"the same id: %s", id))
		}
	}

	return nil
}
