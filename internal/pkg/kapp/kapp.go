package kapp

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/sugarkube/sugarkube/internal/pkg/acquirer"
	"github.com/sugarkube/sugarkube/internal/pkg/convert"
	"github.com/sugarkube/sugarkube/internal/pkg/log"
	"github.com/sugarkube/sugarkube/internal/pkg/vars"
	"gopkg.in/yaml.v2"
)

type installerConfig struct {
	kapp         string
	searchValues []string
	params       map[string]string
}

type Kapp struct {
	id              string
	shouldBePresent bool // if true, this kapp should be present
	// after completing, otherwise it should
	// be absent
	installerConfig installerConfig
	sources         []acquirer.Acquirer
}

const PRESENT_KEY = "present"
const ABSENT_KEY = "absent"
const SOURCES_KEY = "sources"

// Parses kapps and adds them to an array
func parseKapps(kapps *[]Kapp, kappDefinitions map[interface{}]interface{}, shouldBePresent bool) error {

	// parse each kapp definition
	for k, v := range kappDefinitions {
		kapp := Kapp{
			id:              k.(string),
			shouldBePresent: shouldBePresent,
		}

		log.Debugf("kapp=%s, v=%#v", kapp, v)

		// parse the list of sources
		valuesMap, err := convert.MapInterfaceInterfaceToMapStringInterface(v.(map[interface{}]interface{}))
		if err != nil {
			return errors.Wrapf(err, "Error converting manifest value to map")
		}

		// marshal and unmarshal the list of sources
		sourcesBytes, err := yaml.Marshal(valuesMap[SOURCES_KEY])
		if err != nil {
			return errors.Wrapf(err, "Error marshalling sources yaml: %#v", v)
		}

		log.Debugf("Marshalled sources YAML: %s", sourcesBytes)

		sourcesMaps := []map[interface{}]interface{}{}
		err = yaml.UnmarshalStrict(sourcesBytes, &sourcesMaps)
		if err != nil {
			return errors.Wrapf(err, "Error unmarshalling yaml: %s", sourcesBytes)
		}

		log.Debugf("sourcesMaps=%#v", sourcesMaps)

		acquirers := make([]acquirer.Acquirer, 0)
		// now we have a list of sources, get the acquirer for each one
		for _, sourceMap := range sourcesMaps {
			sourceStringMap, err := convert.MapInterfaceInterfaceToMapStringString(sourceMap)
			if err != nil {
				return errors.WithStack(err)
			}

			acquirerImpl, err := acquirer.NewAcquirer(sourceStringMap)
			if err != nil {
				return errors.WithStack(err)
			}

			log.Debugf("Got acquirer %#v", acquirerImpl)

			acquirers = append(acquirers, acquirerImpl)
		}

		kapp.sources = acquirers

		log.Debugf("Parsed kapp=%#v", kapp)

		*kapps = append(*kapps, kapp)
	}

	return nil
}

// Parses manifest YAML data and returns a list of kapps
func parseManifestYaml(data map[string]interface{}) ([]Kapp, error) {
	kapps := make([]Kapp, 0)

	presentKapps, ok := data[PRESENT_KEY]
	if ok {
		err := parseKapps(&kapps, presentKapps.(map[interface{}]interface{}), true)
		if err != nil {
			return nil, errors.Wrap(err, "Error parsing present kapps")
		}
	}

	absentKapps, ok := data[ABSENT_KEY]
	if ok {
		err := parseKapps(&kapps, absentKapps.(map[interface{}]interface{}), false)
		if err != nil {
			return nil, errors.Wrap(err, "Error parsing absent kapps")
		}
	}

	log.Debugf("Parsed kapps to install and remove: %#v", kapps)

	return kapps, nil
}

// Load a single manifest file and parse the kapps it defines
func parseManifestFile(manifestPath string) ([]Kapp, error) {
	log.Debugf("Parsing manifest: %s", manifestPath)

	data, err := vars.LoadYamlFile(manifestPath)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	log.Debugf("Loaded manifest data: %#v", data)

	kapps, err := parseManifestYaml(data)

	return kapps, nil
}

// Parses manifest files and returns a list of kapps on success
func ParseManifests(manifests []string) ([]Kapp, error) {
	log.Debugf("Parsing %d manifest(s)", len(manifests))

	kapps := make([]Kapp, 0)

	for _, manifest := range manifests {
		manifestKapps, err := parseManifestFile(manifest)
		if err != nil {
			return nil, errors.WithStack(err)
		}

		kapps = append(kapps, manifestKapps...)
	}

	return kapps, nil
}

// Validates that the list of kapps doesn't multiple kapps with the same ID, or
// it'll break creating a cache
func ValidateKapps(kapps *[]Kapp) error {
	ids := map[string]bool{}

	for _, kapp := range *kapps {
		id := kapp.id

		if _, ok := ids[id]; ok {
			return errors.New(fmt.Sprintf("Multiple kapps exist with "+
				"the same id: %s", id))
		}
	}

	return nil
}
