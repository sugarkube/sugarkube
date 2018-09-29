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

package kappsot

import (
	"bytes"
	"fmt"
	"github.com/pkg/errors"
	"github.com/sugarkube/sugarkube/internal/pkg/log"
	"github.com/sugarkube/sugarkube/internal/pkg/utils"
	"gopkg.in/yaml.v2"
)

// Uses Helm to determine which kapps are already installed in a target cluster
type HelmKappSot struct {
	charts HelmOutput
}

// Wrapper around Helm output
type HelmOutput struct {
	Next     string
	Releases []HelmRelease
}

const HELM_PATH = "helm"

// struct returned by `helm list --output yaml`
type HelmRelease struct {
	AppVersion string
	Chart      string
	Name       string
	Namespace  string
	Revision   int
	Status     string
	Updated    string
}

// Refreshes the list of Helm charts
func (s HelmKappSot) refresh() error {
	var stdoutBuf, stderrBuf bytes.Buffer

	// todo - add the --kube-context
	err := utils.ExecCommand(HELM_PATH, []string{"list", "--all", "--output", "yaml"},
		map[string]string{}, &stdoutBuf, &stderrBuf, "", 30, false)
	if err != nil {
		return errors.WithStack(err)
	}

	// parse stdout
	output := HelmOutput{}
	err = yaml.Unmarshal(stdoutBuf.Bytes(), &output)
	if err != nil {
		return errors.Wrapf(err, "Error parsing 'Helm list' output: %s",
			stdoutBuf.String())
	}

	s.charts = output

	return nil
}

// Returns whether a helm chart is already successfully installed on the cluster
func (s HelmKappSot) isInstalled(name string, version string) (bool, error) {

	// todo - make sure we refresh this for each manifest to catch the same
	// chart being installed by different manifests accidentally.
	if s.charts.Releases == nil {
		err := s.refresh()
		if err != nil {
			return false, errors.WithStack(err)
		}
	}

	chart := fmt.Sprintf("%s=%s", name, version)

	for _, release := range s.charts.Releases {
		if release.Chart == chart {
			if release.Status == "DEPLOYED" {
				log.Logger.Infof("Chart '%s' is already installed", chart)
				return true, nil
			}

			if release.Status == "FAILED" {
				log.Logger.Infof("The previous release of chart '%s' failed", chart)
				return false, nil
			}

			if release.Status == "DELETED" {
				log.Logger.Infof("Chart '%s' was installed but was deleted", chart)
				return false, nil
			}
		}
	}

	log.Logger.Infof("Chart '%s' isn't installed", chart)

	return false, nil
}
