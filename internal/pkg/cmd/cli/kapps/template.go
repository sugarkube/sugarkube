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
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/sugarkube/sugarkube/internal/pkg/cmd/cli/utils"
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
		Short: fmt.Sprintf("Render templates for kapps"),
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
	f.StringVar(&c.provider, "provider", "", "name of provider, e.g. aws, local, etc.")
	f.StringVar(&c.provisioner, "provisioner", "", "name of provisioner, e.g. kops, minikube, etc.")
	f.StringVar(&c.profile, "profile", "", "launch profile, e.g. dev, test, prod, etc.")
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

	stackConfig, providerImpl, _, err := utils.ProcessCliArgs(c.stackName,
		c.stackFile, cliStackConfig, c.out)
	if err != nil {
		return errors.WithStack(err)
	}

	candidateKapps := map[string]kapp.Kapp{}

	if len(c.includeKapps) > 0 {
		log.Logger.Debugf("Adding %d kapps to the candidate template set", len(c.includeKapps))
		candidateKapps, err = getKappsByFullyQualifiedId(c.includeKapps, stackConfig)
		if err != nil {
			return errors.WithStack(err)
		}
	} else {
		log.Logger.Debugf("Adding all kapps to the candidate template set")

		log.Logger.Debugf("Stack config has %d manifests", len(stackConfig.AllManifests()))

		// select all kapps
		for _, manifest := range stackConfig.AllManifests() {
			log.Logger.Debugf("Manifest '%s' contains %d kapps", manifest.Id, len(manifest.Kapps))

			for _, manifestKapp := range manifest.Kapps {
				candidateKapps[manifestKapp.FullyQualifiedId()] = manifestKapp
			}
		}
	}

	log.Logger.Debugf("There are %d candidate kapps for templating (before applying exclusions)",
		len(candidateKapps))

	if len(c.excludeKapps) > 0 {
		// delete kapps
		excludedKapps, err := getKappsByFullyQualifiedId(c.excludeKapps, stackConfig)
		if err != nil {
			return errors.WithStack(err)
		}

		log.Logger.Debugf("Excluding %d kapps from the templating set", len(excludedKapps))

		for k, _ := range excludedKapps {
			if _, ok := candidateKapps[k]; ok {
				delete(candidateKapps, k)
			}
		}
	}

	_, err = fmt.Fprintf(c.out, "Rendering templates for %d kapps\n", len(candidateKapps))
	if err != nil {
		return errors.WithStack(err)
	}

	err = RenderTemplates(candidateKapps, c.cacheDir, stackConfig, providerImpl,
		c.dryRun)
	if err != nil {
		return errors.WithStack(err)
	}

	_, err = fmt.Fprintln(c.out, "Templates successfully rendered")
	if err != nil {
		return errors.WithStack(err)
	}

	return nil
}

// Render templates for kapps defined in a stack config
func RenderTemplates(kapps map[string]kapp.Kapp, cacheDir string,
	stackConfig *kapp.StackConfig, providerImpl provider.Provider, dryRun bool) error {

	if len(kapps) == 0 {
		return errors.New("No kapps supplied to template function")
	}

	// make sure the cache dir exists if set
	if cacheDir != "" {
		if _, err := os.Stat(cacheDir); err != nil {
			return errors.New(fmt.Sprintf("Cache dir '%s' doesn't exist",
				cacheDir))
		}
	}

	candidateKappIds := []string{}
	for k, _ := range kapps {
		candidateKappIds = append(candidateKappIds, k)
	}

	log.Logger.Debugf("Rendering templates for kapps: %s", strings.Join(candidateKappIds, ", "))

	for _, kappObj := range kapps {
		mergedKappVars, err := templater.MergeVarsForKapp(&kappObj, stackConfig, &providerImpl)
		err = templateKapp(&kappObj, mergedKappVars, stackConfig, cacheDir, dryRun)
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

		for _, manifest := range stackConfig.AllManifests() {
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

// Render templates for an individual kapp
func templateKapp(kappObj *kapp.Kapp, mergedKappVars map[string]interface{},
	stackConfig *kapp.StackConfig, cacheDir string, dryRun bool) error {

	var err error

	if len(kappObj.Templates) == 0 {
		log.Logger.Debugf("No templates to render for kapp '%s'",
			kappObj.FullyQualifiedId())
		return nil
	}

	kappObj.SetCacheDir(cacheDir)

	log.Logger.Infof("Rendering templates for kapp '%s'",
		kappObj.FullyQualifiedId())

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

		log.Logger.Debugf("Templating file '%s' with vars: %#v", templateSource, mergedKappVars)

		destPath := templateDefinition.Dest
		if !filepath.IsAbs(destPath) {
			destPath = filepath.Join(kappObj.CacheDir(), destPath)
		}

		// check whether the dest path exists
		if _, err := os.Stat(destPath); err == nil {
			log.Logger.Infof("Template destination path '%s' exists. "+
				"File will be overwritten by rendered template '%s' for kapp '%s'",
				destPath, templateSource, kappObj.Id)
		}

		// check whether the parent directory for dest path exists and return an error if not
		destDir := filepath.Dir(destPath)
		if _, err := os.Stat(destDir); os.IsNotExist(err) {
			return errors.New(fmt.Sprintf("Can't write template to non-existent directory: %s", destDir))
		}

		var outBuf bytes.Buffer

		err = templater.TemplateFile(templateSource, &outBuf, mergedKappVars)
		if err != nil {
			return errors.WithStack(err)
		}

		if dryRun {
			log.Logger.Infof("Dry run. Template '%s' for kapp '%s' which "+
				"would be written to '%s' rendered as:\n%s", templateSource,
				kappObj.Id, destPath, outBuf.String())
		} else {
			log.Logger.Infof("Writing rendered template '%s' for kapp "+
				"'%s' to '%s'", templateSource, kappObj.FullyQualifiedId(), destPath)
			err := ioutil.WriteFile(destPath, outBuf.Bytes(), 0644)
			if err != nil {
				return errors.WithStack(err)
			}
		}
	}

	return nil
}
