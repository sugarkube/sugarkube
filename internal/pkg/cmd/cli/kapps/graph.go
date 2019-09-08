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

package kapps

import (
	"bytes"
	"fmt"
	"github.com/pkg/errors"
	"github.com/skratchdot/open-golang/open"
	"github.com/spf13/cobra"
	"github.com/sugarkube/sugarkube/internal/pkg/constants"
	"github.com/sugarkube/sugarkube/internal/pkg/log"
	"github.com/sugarkube/sugarkube/internal/pkg/printer"
	"github.com/sugarkube/sugarkube/internal/pkg/stack"
	"github.com/sugarkube/sugarkube/internal/pkg/structs"
	"github.com/sugarkube/sugarkube/internal/pkg/utils"
	"io/ioutil"
	"os"
)

type graphCmd struct {
	workspaceDir    string
	includeParents  bool
	open            bool
	outPath         string
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

func newGraphCmd() *cobra.Command {
	c := &graphCmd{}

	cmd := &cobra.Command{
		Use:   "graph [flags] [stack-file] [stack-name]",
		Short: fmt.Sprintf("Graphs local kapps"),
		Long: `Prints the graph showing which kapps would be processed.

The graph can also optionally be rendered as an SVG image.
`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) < 2 {
				return errors.New("some required arguments are missing")
			} else if len(args) > 2 {
				return errors.New("too many arguments supplied")
			}
			c.stackFile = args[0]
			c.stackName = args[1]

			return c.run()
		},
	}

	f := cmd.Flags()
	f.BoolVar(&c.open, "open", false, "produce an SVG visualisation and open it in the default .svg application (required graphviz)")
	f.StringVarP(&c.outPath, "out", "o", "", "write an SVG visualisation to the given file path (required graphviz)")
	f.BoolVar(&c.includeParents, "parents", false, "process all parents of all selected kapps as well")
	f.StringVar(&c.provider, "provider", "", "name of provider, e.g. aws, local, etc.")
	f.StringVar(&c.provisioner, "provisioner", "", "name of provisioner, e.g. kops, minikube, etc.")
	f.StringVar(&c.profile, "profile", "", "launch profile, e.g. dev, test, prod, etc.")
	f.StringVarP(&c.cluster, "cluster", "c", "", "name of cluster to launch, e.g. dev1, dev2, etc.")
	f.StringVarP(&c.account, "account", "a", "", "string identifier for the account to launch in (for providers that support it)")
	f.StringVarP(&c.region, "region", "r", "", "name of region (for providers that support it)")
	f.StringArrayVarP(&c.includeSelector, "include", "i", []string{},
		fmt.Sprintf("only process specified kapps (can specify multiple, formatted 'manifest-id:kapp-id' or 'manifest-id:%s' for all)",
			constants.WildcardCharacter))
	f.StringArrayVarP(&c.excludeSelector, "exclude", "x", []string{},
		fmt.Sprintf("exclude individual kapps (can specify multiple, formatted 'manifest-id:kapp-id' or 'manifest-id:%s' for all)",
			constants.WildcardCharacter))
	return cmd
}

func (c *graphCmd) run() error {

	// CLI overrides - will be merged with any loaded from a stack config file
	cliStackConfig := &structs.StackFile{
		Provider:    c.provider,
		Provisioner: c.provisioner,
		Profile:     c.profile,
		Cluster:     c.cluster,
		Region:      c.region,
		Account:     c.account,
	}

	var err error

	stackObj, err = stack.BuildStack(c.stackName, c.stackFile, cliStackConfig)
	if err != nil {
		return errors.WithStack(err)
	}

	dagObj, err := BuildDagForSelected(stackObj, c.workspaceDir, c.includeSelector, c.excludeSelector, c.includeParents)
	if err != nil {
		return errors.WithStack(err)
	}

	_, err = printer.Fprintln("")
	if err != nil {
		return errors.WithStack(err)
	}

	if c.open || c.outPath != "" {
		log.Logger.Debugf("Generating graphViz definition...")
		graphViz := dagObj.Visualise(stackObj.GetConfig().GetName())

		// write the graphViz config to a file
		dotFile, err := ioutil.TempFile("", "sugarkube-svg-")
		if err != nil {
			return errors.WithStack(err)
		}

		log.Logger.Debugf("Writing graphViz file to: %s", dotFile.Name())

		_, err = dotFile.Write([]byte(graphViz))
		if err != nil {
			return errors.WithStack(err)
		}
		err = dotFile.Close()
		if err != nil {
			return errors.WithStack(err)
		}

		var outFile *os.File

		if c.outPath != "" {
			outFile, err = os.OpenFile(c.outPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0666)
			if err != nil {
				return errors.WithStack(err)
			}
		} else {
			outFile, err = ioutil.TempFile("", "sugarkube-graph.*.svg")
			if err != nil {
				return errors.WithStack(err)
			}
		}

		var stdoutBuf, stderrBuf bytes.Buffer
		workingDir, err := os.Getwd()
		if err != nil {
			return errors.WithStack(err)
		}

		err = utils.ExecCommand("dot", []string{"-Tsvg", dotFile.Name(), "-o", outFile.Name()},
			map[string]string{}, &stdoutBuf, &stderrBuf, workingDir, 5, 0, false)
		if err != nil {
			return errors.WithStack(err)
		}

		_, err = printer.Fprintf("[green]SVG written to %s!\n", outFile.Name())
		if err != nil {
			return errors.WithStack(err)
		}

		if c.open {
			err = open.Start(outFile.Name())
			if err != nil {
				return errors.WithStack(err)
			}
		}
	}

	return nil
}
