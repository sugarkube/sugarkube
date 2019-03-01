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
	"github.com/sugarkube/sugarkube/internal/pkg/templater"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

type templateConfig struct {
	out             io.Writer
	dryRun          bool
	cacheDir        string
	stackName       string
	stackFile       string
	provider        string
	provisioner     string
	profile         string
	account         string
	cluster         string
	region          string
	includeSelector []string
	excludeSelector []string
}

func newTemplateCmd(out io.Writer) *cobra.Command {
	c := &templateConfig{
		out: out,
	}

	cmd := &cobra.Command{
		Use:   "template [flags] [stack-file] [stack-name] [cache-dir]",
		Short: fmt.Sprintf("Render templates for kapps"),
		Long: `Renders configured templates for kapps, useful for e.g. terraform backends 
configured for the region the target cluster is in, generating Helm 
'values.yaml' files, etc.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) < 3 {
				return errors.New("some required arguments are missing")
			} else if len(args) > 3 {
				return errors.New("too many arguments supplied")
			}
			c.stackFile = args[0]
			c.stackName = args[1]
			c.cacheDir = args[2]
			return c.run()
		},
	}

	f := cmd.Flags()
	f.BoolVarP(&c.dryRun, "dry-run", "n", false, "show what would happen but don't create a cluster")
	f.StringVar(&c.provider, "provider", "", "name of provider, e.g. aws, local, etc.")
	f.StringVar(&c.provisioner, "provisioner", "", "name of provisioner, e.g. kops, minikube, etc.")
	f.StringVar(&c.profile, "profile", "", "launch profile, e.g. dev, test, prod, etc.")
	f.StringVarP(&c.cluster, "cluster", "c", "", "name of cluster to launch, e.g. dev1, dev2, etc.")
	f.StringVarP(&c.account, "account", "a", "", "string identifier for the account to launch in (for providers that support it)")
	f.StringVarP(&c.region, "region", "r", "", "name of region (for providers that support it)")
	f.StringArrayVarP(&c.includeSelector, "include", "i", []string{},
		fmt.Sprintf("only process specified kapps (can specify multiple, formatted manifest-id:kapp-id or 'manifest-id:%s' for all)",
			kapp.WILDCARD_CHARACTER))
	f.StringArrayVarP(&c.excludeSelector, "exclude", "x", []string{},
		fmt.Sprintf("exclude individual kapps (can specify multiple, formatted manifest-id:kapp-id or 'manifest-id:%s' for all)",
			kapp.WILDCARD_CHARACTER))
	return cmd
}

func (c *templateConfig) run() error {

	// CLI overrides - will be merged with any loaded from a stack config file
	cliStackConfig := &kapp.StackConfig{
		Provider:    c.provider,
		Provisioner: c.provisioner,
		Profile:     c.profile,
		Cluster:     c.cluster,
		Region:      c.region,
		Account:     c.account,
	}

	stackConfig, err := utils.BuildStackConfig(c.stackName, c.stackFile, cliStackConfig, c.out)
	if err != nil {
		return errors.WithStack(err)
	}

	selectedKapps, err := kapp.SelectKapps(stackConfig.AllManifests(), c.includeSelector, c.excludeSelector)
	if err != nil {
		return errors.WithStack(err)
	}

	_, err = fmt.Fprintf(c.out, "Rendering templates for %d kapps\n", len(selectedKapps))
	if err != nil {
		return errors.WithStack(err)
	}

	err = RenderTemplates(selectedKapps, c.cacheDir, stackConfig, c.dryRun)
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
	stackConfig *kapp.StackConfig, dryRun bool) error {

	if len(kapps) == 0 {
		return errors.New("No kapps supplied to template function")
	}

	// make sure the cache dir exists
	if _, err := os.Stat(cacheDir); err != nil {
		return errors.New(fmt.Sprintf("Cache dir '%s' doesn't exist",
			cacheDir))
	}

	candidateKappIds := make([]string, 0)
	for k := range kapps {
		candidateKappIds = append(candidateKappIds, k)
	}

	log.Logger.Debugf("Rendering templates for kapps: %s", strings.Join(candidateKappIds, ", "))

	for _, kappObj := range kapps {
		mergedKappVars, err := kapp.MergeVarsForKapp(&kappObj, stackConfig, map[string]interface{}{})
		if err != nil {
			return errors.WithStack(err)
		}
		err = templateKapp(&kappObj, mergedKappVars, stackConfig, cacheDir, dryRun)
		if err != nil {
			return errors.WithStack(err)
		}
	}

	return nil
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
