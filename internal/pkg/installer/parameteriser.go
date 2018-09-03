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

package installer

import (
	"github.com/pkg/errors"
	"github.com/sugarkube/sugarkube/internal/pkg/kapp"
	"github.com/sugarkube/sugarkube/internal/pkg/provider"
	"os"
	"os/user"
	"path/filepath"
	"strings"
)

// This is a generic way of inspecting kapps to see what they contain and what
// env vars/CLI parameters should be passed to their installers

// todo - move into a default config file, but allow them to be overridden/
// addtional interfaces defined.
//var parameteriserConfig = `
//kapp_interfaces:	# different things a kapp might contain. A kapp may 'implement'
//					# multiple interfaces (e.g. contain both a helm chart and terraform configs
//  helm_chart:
//    heuristics: 			# inspections we can carry out on a kapp to see what it contains
//    - file:
//        pattern: Chart.yaml		# regex to search for under the kapp root dir
//    params:
//      env:
//      - name: KUBE_CONTEXT
//        value:
//          type: vars_lookup
//          path: provider
//          key: kube_context
//      - name: NAMESPACE		# default value. Allow overriding it in the installer config.
//        value: 				# think of how to configure that here...
//          type: obj_field
//          path: kapp
//          key: Id
//      - name: RELEASE
//        value:
//          type: obj_field
//          path: kapp
//          key: Id
//      cliArgs:
//      - name: helm-opts
//        components:
//        - key: -f
//          value
//            pattern: values-(\w+).yaml
//
//  k8s_resource:             # a naked k8s resource. No heuristics. Expect to find
//    params:					# it listed in 'sugarkube.yaml'
//      env:
//      - name: KUBE_CONTEXT
//        value:
//          type: vars_lookup
//          path: provider
//          key: kube_context
//
//  terraform:
//    heuristics:
//    - file:
//        pattern: terraform.*
//        type: dir
//    params:
//      cliArgs:
//      - name: tf-opts
//        components:			# by default collapse multiple values into a
//        - key: -var-file		# single CLI arg
//          value:
//            pattern: vars/(\w+).tfvars
//`

const IMPLEMENTS_HELM = "helm"
const IMPLEMENTS_TERRAFORM = "terraform"
const IMPLEMENTS_K8S = "k8s"

type Parameteriser struct {
	Name    string
	kappObj *kapp.Kapp
}

const KUBE_CONTEXT_KEY = "kube_context"

// Return a map of env vars that should be passed to the kapp by the installer
func (i *Parameteriser) GetEnvVars(vars provider.Values) (map[string]string, error) {
	envVars := make(map[string]string)

	if i.Name == IMPLEMENTS_HELM {
		envVars["NAMESPACE"] = i.kappObj.Id
		envVars["RELEASE"] = i.kappObj.Id

		// todo - this is a hack. Need to find a place for things like this and
		// a generic way of adding them as env vars. Perhaps have a YAML block
		// called 'installers' or something?
		envVars["HOSTED_ZONE"] = vars["hosted_zone"].(string)
	}

	if i.Name == IMPLEMENTS_HELM || i.Name == IMPLEMENTS_K8S {
		if kubeContext, ok := os.LookupEnv("KUBE_CONTEXT"); ok {
			// todo - we probably don't need this... remove if not
			envVars["KUBE_CONTEXT"] = kubeContext
		} else {
			envVars["KUBE_CONTEXT"] = vars[KUBE_CONTEXT_KEY].(string)
		}

		// todo - we probably don't need this... remove if not
		// only set env var if it's not already set
		if kubeConfig, ok := os.LookupEnv("KUBECONFIG"); ok {
			envVars["KUBECONFIG"] = kubeConfig
		} else {
			usr, _ := user.Current()
			homeDir := usr.HomeDir
			defaultKubeConfig := filepath.Join(homeDir, ".kube/config")
			envVars["KUBECONFIG"] = defaultKubeConfig
		}
	}

	return envVars, nil
}

// Returns a list of args that the installer should pass to the kapp. This will
// need refactoring once parsing the Parameteriser config is implemented.
func (i *Parameteriser) GetCliArgs(validPatternMatches []string) (string, error) {
	pattern := ""
	argName := ""
	argKey := ""

	if i.Name == IMPLEMENTS_HELM {
		pattern = "values-(?P<Var>\\w*).yaml"
		argName = "helm-opts"
		argKey = "-f"
	}

	if i.Name == IMPLEMENTS_TERRAFORM {
		pattern = "terraform.*"
		argName = "tf-opts"
		argKey = "-var-file"
	}

	if pattern == "" {
		return "", nil
	}

	matches, err := findFilesByPattern(i.kappObj.RootDir, pattern,
		true, true)
	if err != nil {
		return "", errors.WithStack(err)
	}

	// use a map for deduping
	argValues := make(map[string]string, 0)

	// make sure the matching group in each match is in the valid pattern matches list
	for _, match := range matches {
		matchingGroups := getRegExpCapturingGroups(pattern, match)

		// don't punish yourself by saying the words "functional programming"...
		for _, v := range matchingGroups {
			for _, valid := range validPatternMatches {
				if v == valid {
					// todo - this separator string may need to be configurable too
					argValues[match] = strings.Join([]string{argKey, match}, " ")
				}
			}
		}
	}

	cliArg := ""

	if len(argValues) > 0 {
		strArgs := make([]string, 0)

		for _, v := range argValues {
			strArgs = append(strArgs, v)
		}

		joinedValues := strings.Join(strArgs, " ")
		cliArg = strings.Join([]string{argName, joinedValues}, "=")
	}
	return cliArg, nil
}

// Examines a kapp to find out what it contains, and therefore what env vars/
// CLI args need passing to it by an Installer.
func identifyKappInterfaces(kappObj *kapp.Kapp) ([]Parameteriser, error) {
	// todo - parse the above config and test the kapp using it.
	// todo - also look in the kapp's sugarkube.yaml file if it exists

	parameterisers := make([]Parameteriser, 0)

	// todo - remove IMPLEMENTS_K8S from this. It's a temporary kludge until we
	// can get it from the kapp's sugarkube.yaml file
	parameterisers = append(parameterisers, Parameteriser{
		Name: IMPLEMENTS_K8S, kappObj: kappObj})

	// todo - remove this kludge to find out whether the kapp contains a helm chart.
	chartPaths, err := findFilesByPattern(kappObj.RootDir, "Chart.yaml",
		true, true)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	if len(chartPaths) > 0 {
		parameterisers = append(parameterisers, Parameteriser{
			Name: IMPLEMENTS_HELM, kappObj: kappObj})
	}

	// todo - remove this kludge to find out whether the kapp contains terraform configs
	terraformPaths, err := findFilesByPattern(kappObj.RootDir, "terraform",
		true, true)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	if len(terraformPaths) > 0 {
		parameterisers = append(parameterisers, Parameteriser{
			Name: IMPLEMENTS_TERRAFORM, kappObj: kappObj})
	}

	return parameterisers, nil
}
