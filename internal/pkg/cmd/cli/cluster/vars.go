/*
 * Copyright 2019 The Sugarkube Authors
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

package cluster

import (
	"fmt"
	"github.com/imdario/mergo"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/sugarkube/sugarkube/internal/pkg/config"
	"github.com/sugarkube/sugarkube/internal/pkg/log"
	"github.com/sugarkube/sugarkube/internal/pkg/stack"
	"github.com/sugarkube/sugarkube/internal/pkg/structs"
	datautils "github.com/sugarkube/sugarkube/internal/pkg/utils"
	"gopkg.in/yaml.v2"
	"io"
	"strings"
)

type varsConfig struct {
	out         io.Writer
	cacheDir    string
	stackName   string
	stackFile   string
	provider    string
	provisioner string
	profile     string
	account     string
	cluster     string
	region      string
	suppress    []string
}

func newVarsCmd(out io.Writer) *cobra.Command {
	c := &varsConfig{
		out: out,
	}

	cmd := &cobra.Command{
		Use:   "vars [flags] [stack-file] [stack-name]",
		Short: fmt.Sprintf("Display all variables available for a stack"),
		Long:  `Merges variables from all sources and displays them.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) < 2 {
				return errors.New("the name of the stack to run, and the path to the stack file are required")
			} else if len(args) > 2 {
				return errors.New("too many arguments supplied")
			}
			c.stackFile = args[0]
			c.stackName = args[1]
			return c.run()
		},
	}

	f := cmd.Flags()
	f.StringVar(&c.provider, "provider", "", "name of provider, e.g. aws, local, etc.")
	f.StringVar(&c.provisioner, "provisioner", "", "name of provisioner, e.g. kops, minikube, etc.")
	f.StringVar(&c.profile, "profile", "", "launch profile, e.g. dev, test, prod, etc.")
	f.StringVarP(&c.cluster, "cluster", "c", "", "name of cluster to launch, e.g. dev1, dev2, etc.")
	f.StringVarP(&c.account, "account", "a", "", "string identifier for the account to launch in (for providers that support it)")
	f.StringVarP(&c.region, "region", "r", "", "name of region (for providers that support it)")
	f.StringArrayVarP(&c.suppress, "suppress", "s", []string{},
		"paths to variables to suppress from the output to simplify it (e.g. 'provision.specs')")
	return cmd
}

func (c *varsConfig) run() error {

	// CLI overrides - will be merged with any loaded from a stack config file
	cliStackConfig := &structs.Stack{
		Provider:    c.provider,
		Provisioner: c.provisioner,
		Profile:     c.profile,
		Cluster:     c.cluster,
		Region:      c.region,
		Account:     c.account,
	}

	stackObj, err := stack.BuildStack(c.stackName, c.stackFile, cliStackConfig,
		config.CurrentConfig, c.out)
	if err != nil {
		return errors.WithStack(err)
	}

	_, err = fmt.Fprintf(c.out, "Displaying variables for stack '%s':\n\n",
		stackObj.Config.Name())
	if err != nil {
		return errors.WithStack(err)
	}

	templatedVars, err := stackObj.TemplatedVars(nil, map[string]interface{}{})
	if err != nil {
		return errors.WithStack(err)
	}

	if len(c.suppress) > 0 {
		for _, exclusion := range c.suppress {
			// trim any leading zeroes for compatibility with how variables are referred to in templates
			exclusion = strings.TrimPrefix(exclusion, ".")
			blanked := datautils.BlankNestedMap(map[string]interface{}{}, strings.Split(exclusion, "."))
			log.Logger.Debugf("blanked=%#v", blanked)

			err = mergo.Merge(&templatedVars, blanked, mergo.WithOverride)
			if err != nil {
				return errors.WithStack(err)
			}
		}
	}

	yamlData, err := yaml.Marshal(&templatedVars)
	if err != nil {
		return errors.WithStack(err)
	}

	_, err = fmt.Fprint(c.out, string(yamlData))
	if err != nil {
		return errors.WithStack(err)
	}

	return nil
}
