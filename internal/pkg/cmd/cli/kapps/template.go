/*
 * Copyright 2018 The Sugarkube Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package kapps

import (
	"bytes"
	"fmt"
	"github.com/imdario/mergo"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/sugarkube/sugarkube/internal/pkg/cmd/cli/utils"
	"github.com/sugarkube/sugarkube/internal/pkg/convert"
	"github.com/sugarkube/sugarkube/internal/pkg/kapp"
	"github.com/sugarkube/sugarkube/internal/pkg/log"
	"github.com/sugarkube/sugarkube/internal/pkg/provider"
	"github.com/sugarkube/sugarkube/internal/pkg/templater"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

type templateConfig struct {
	out         io.Writer
	dryRun      bool
	cacheDir    string
	stackName   string
	stackFile   string
	provider    string
	provisioner string
	//kappVarsDirs cmd.Files
	profile string
	account string
	cluster string
	region  string
	//manifests    cmd.Files
	includeKapps []string
	excludeKapps []string
}

func newTemplateCmd(out io.Writer) *cobra.Command {
	c := &templateConfig{
		out: out,
	}

	cmd := &cobra.Command{
		Use:   "template [flags]",
		Short: fmt.Sprintf("Generate templates for kapps"),
		Long: `Renders configured templates for kapps, useful for e.g. terraform backends 
configured for the region the target cluster is in, generating Helm 
'values.yaml' files, etc.`,
		RunE: c.run,
	}

	f := cmd.Flags()
	f.BoolVar(&c.dryRun, "dry-run", false, "show what would happen but don't create a cluster")
	f.StringVarP(&c.stackName, "stack-name", "n", "", "name of a stack to launch (required when passing --stack-config)")
	f.StringVarP(&c.stackFile, "stack-config", "s", "", "path to file defining stacks by name")
	f.StringVarP(&c.cacheDir, "dir", "d", "", "Directory containing the kapp cache to write rendered templates to")
	f.StringVarP(&c.provider, "provider", "p", "", "name of provider, e.g. aws, local, etc.")
	f.StringVarP(&c.provisioner, "provisioner", "v", "", "name of provisioner, e.g. kops, minikube, etc.")
	f.StringVarP(&c.profile, "profile", "l", "", "launch profile, e.g. dev, test, prod, etc.")
	f.StringVarP(&c.cluster, "cluster", "c", "", "name of cluster to launch, e.g. dev1, dev2, etc.")
	f.StringVarP(&c.account, "account", "a", "", "string identifier for the account to launch in (for providers that support it)")
	f.StringVarP(&c.region, "region", "r", "", "name of region (for providers that support it)")
	f.StringArrayVarP(&c.includeKapps, "include", "i", []string{}, "only process specified kapps (can specify multiple, formatted manifest-id:kapp-id)")
	f.StringArrayVarP(&c.excludeKapps, "exclude", "x", []string{}, "exclude individual kapps (can specify multiple, formatted manifest-id:kapp-id)")
	// these are commented for now to keep things simple, but ultimately we should probably support taking these as CLI args
	//f.VarP(&c.kappVarsDirs, "dir", "f", "Paths to YAML directory to load kapp values from (can specify multiple)")
	//f.VarP(&c.manifests, "manifest", "m", "YAML manifest file to load (can specify multiple but will replace any configured in a stack)")
	return cmd
}

func (c *templateConfig) run(cmd *cobra.Command, args []string) error {

	// CLI overrides - will be merged with any loaded from a stack config file
	cliStackConfig := &kapp.StackConfig{
		Provider:    c.provider,
		Provisioner: c.provisioner,
		Profile:     c.profile,
		Cluster:     c.cluster,
		Region:      c.region,
		Account:     c.account,
		//KappVarsDirs: c.kappVarsDirs,
		//Manifests:    cliManifests,
	}

	// make sure the cache dir exists if set
	if c.cacheDir != "" {
		if _, err := os.Stat(c.cacheDir); err != nil {
			return errors.New(fmt.Sprintf("Cache dir '%s' doesn't exist",
				c.cacheDir))
		}
	}

	stackConfig, providerImpl, _, err := utils.ProcessCliArgs(c.stackName,
		c.stackFile, cliStackConfig)
	if err != nil {
		return errors.WithStack(err)
	}

	candidateKapps := map[string]kapp.Kapp{}

	if len(c.includeKapps) > 0 {
		candidateKapps, err = getKappsByFullyQualifiedId(c.includeKapps, stackConfig)
		if err != nil {
			return errors.WithStack(err)
		}
	} else {
		// select all kapps
		for _, manifest := range stackConfig.Manifests {
			for _, manifestKapp := range manifest.Kapps {
				fqId := fmt.Sprintf("%s:%s", manifest.Id, manifestKapp.Id)
				candidateKapps[fqId] = manifestKapp
			}
		}
	}

	if len(c.excludeKapps) > 0 {
		// delete kapps
		excludedKapps, err := getKappsByFullyQualifiedId(c.excludeKapps, stackConfig)
		if err != nil {
			return errors.WithStack(err)
		}

		for k, _ := range excludedKapps {
			if _, ok := candidateKapps[k]; ok {
				delete(candidateKapps, k)
			}
		}
	}

	candidateKappIds := []string{}
	for k, _ := range candidateKapps {
		candidateKappIds = append(candidateKappIds, k)
	}

	log.Logger.Debugf("Templating candidate kapps: %s", strings.Join(candidateKappIds, ", "))

	stackConfigMap := stackConfig.AsMap()
	// convert the map to the appropriate type
	namespacedStackConfigMap := map[string]interface{}{
		"stack": convert.MapStringStringToMapStringInterface(stackConfigMap),
	}

	providerVars := provider.GetVars(providerImpl)

	for _, kappObj := range candidateKapps {
		err = templateKapp(&kappObj, stackConfig, namespacedStackConfigMap,
			providerVars, c.cacheDir, c.dryRun)
		if err != nil {
			return errors.WithStack(err)
		}
	}

	return nil
}

// Returns kapps from a stack config by fully-qualified ID, i.e. `manifest-id:kapp-id`
func getKappsByFullyQualifiedId(kapps []string, stackConfig *kapp.StackConfig) (map[string]kapp.Kapp, error) {
	results := map[string]kapp.Kapp{}

	for _, fqKappId := range kapps {
		splitKappId := strings.Split(fqKappId, ":")

		if len(splitKappId) != 2 {
			return nil, errors.New("Fully-qualified kapps must be given, i.e. " +
				"formatted 'manifest-id:kapp-id'")
		}

		manifestId := splitKappId[0]
		kappId := splitKappId[1]

		for _, manifest := range stackConfig.Manifests {
			if manifestId != manifest.Id {
				continue
			}

			for _, manifestKapp := range manifest.Kapps {
				if manifestKapp.Id == kappId {
					results[fqKappId] = manifestKapp
				}
			}
		}
	}

	return results, nil
}

func templateKapp(kappObj *kapp.Kapp, stackConfig *kapp.StackConfig,
	stackConfigMap map[string]interface{}, providerVarsMap map[string]interface{},
	cacheDir string, dryRun bool) error {

	kappObj.SetCacheDir(cacheDir)

	log.Logger.Debugf("Rendering templates for kapp '%s'", kappObj.Id)

	mergedVars := map[string]interface{}{}
	err := mergo.Merge(&mergedVars, stackConfigMap, mergo.WithOverride)
	if err != nil {
		return errors.WithStack(err)
	}

	err = mergo.Merge(&mergedVars, providerVarsMap, mergo.WithOverride)
	if err != nil {
		return errors.WithStack(err)
	}

	kappMap := kappObj.AsMap()
	kappVars, err := stackConfig.GetKappVars(kappObj)
	if err != nil {
		return errors.WithStack(err)
	}

	namespacedKappMap := map[string]interface{}{
		"kapp": convert.MapStringStringToMapStringInterface(kappMap),
	}
	err = mergo.Merge(&mergedVars, namespacedKappMap, mergo.WithOverride)
	if err != nil {
		return errors.WithStack(err)
	}

	err = mergo.Merge(&mergedVars, kappVars, mergo.WithOverride)
	if err != nil {
		return errors.WithStack(err)
	}

	for _, templateDefinition := range kappObj.Templates {
		templateSource := templateDefinition.Source
		if !filepath.IsAbs(templateSource) {
			foundTemplate := false

			// search each template directory defined in the stack config
			for _, templateDir := range stackConfig.TemplateDirs {
				possibleSource := filepath.Join(stackConfig.Dir(), templateDir, templateSource)
				_, err := os.Stat(possibleSource)
				if err == nil {
					templateSource = possibleSource
					foundTemplate = true
					break
				}
			}

			if !foundTemplate {
				return errors.New(fmt.Sprintf("Failed to find template '%s' "+
					"in any of the defined template directories: %s", templateSource,
					strings.Join(stackConfig.TemplateDirs, ", ")))
			}
		}

		if !filepath.IsAbs(templateSource) {
			templateSource, err = filepath.Abs(templateSource)
			if err != nil {
				return errors.WithStack(err)
			}
		}

		log.Logger.Debugf("Templating file '%s' with vars: %#v", templateSource, mergedVars)

		destPath := templateDefinition.Dest
		if !filepath.IsAbs(destPath) {
			destPath = filepath.Join(kappObj.CacheDir(), destPath)
		}

		// check whether the dest path exists
		if _, err := os.Stat(destPath); err == nil {
			log.Logger.Infof("Rendering template '%s' for kapp '%s'. "+
				"Destination path '%s' already exists. File will be overwritten.",
				templateSource, kappObj.Id, destPath)
		} else {
			log.Logger.Infof("Rendering template '%s' for kapp '%s' to %s ",
				templateSource, kappObj.Id, destPath)
		}

		var outBuf bytes.Buffer

		err = templater.TemplateFile(templateSource, &outBuf, mergedVars)
		if err != nil {
			return errors.WithStack(err)
		}

		if dryRun {
			log.Logger.Infof("Dry run. Template '%s' for kapp '%s' which "+
				"would be written to '%s' rendered as:\n%s", templateSource,
				kappObj.Id, destPath, outBuf.String())
		} else {
			log.Logger.Debugf("Writing rendered template '%s' for kapp "+
				"'%s' to '%s'", templateSource, kappObj.Id, destPath)
			err := ioutil.WriteFile(destPath, outBuf.Bytes(), 0644)
			if err != nil {
				return errors.WithStack(err)
			}
		}
	}

	return nil
}
