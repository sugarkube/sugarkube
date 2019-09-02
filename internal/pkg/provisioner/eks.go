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

package provisioner

import (
	"bytes"
	"context"
	"fmt"
	"github.com/pkg/errors"
	"github.com/sugarkube/sugarkube/internal/pkg/clustersot"
	"github.com/sugarkube/sugarkube/internal/pkg/constants"
	"github.com/sugarkube/sugarkube/internal/pkg/interfaces"
	"github.com/sugarkube/sugarkube/internal/pkg/log"
	"github.com/sugarkube/sugarkube/internal/pkg/printer"
	"github.com/sugarkube/sugarkube/internal/pkg/utils"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os/exec"
)

const eksProvisionerName = "eks"
const eksDefaultBinary = "eksctl"

const eksCommandTimeoutSeconds = 30
const eksCommandTimeoutSecondsLong = 300

// number of seconds to sleep after the cluster has come online before checking whether
// it's ready
const eksSleepSecondsBeforeReadyCheck = 60

// todo - catch errors accessing these
const configKeyEKSClusterName = "name"

type EksProvisioner struct {
	clusterSot           interfaces.IClusterSot
	stack                interfaces.IStack
	eksConfig            EksConfig
	portForwardingActive bool
	sshCommand           *exec.Cmd
}

type EksConfig struct {
	clusterName string // set after parsing the eks YAML
	Binary      string // path to the eksctl binary
	Params      struct {
		Global        map[string]string
		GetCluster    map[string]string      `yaml:"get_cluster"`
		CreateCluster map[string]string      `yaml:"create_cluster"`
		DeleteCluster map[string]string      `yaml:"delete_cluster"`
		UpdateCluster map[string]string      `yaml:"update_cluster"`
		ConfigFile    map[string]interface{} `yaml:"config_file"`
	}
}

// Instantiates a new instance
func newEksProvisioner(stackConfig interfaces.IStack, clusterSot interfaces.IClusterSot) (*EksProvisioner, error) {
	eksConfig, err := parseEksConfig(stackConfig)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return &EksProvisioner{
		stack:      stackConfig,
		eksConfig:  *eksConfig,
		clusterSot: clusterSot,
	}, nil
}

func (p EksProvisioner) GetStack() interfaces.IStack {
	return p.stack
}

func (p EksProvisioner) ClusterSot() interfaces.IClusterSot {
	return p.clusterSot
}

// Returns a bool indicating whether the cluster exists (but it may not yet respond to kubectl commands)
func (p EksProvisioner) clusterExists() (bool, error) {

	templatedVars, err := p.stack.GetTemplatedVars(nil, map[string]interface{}{})
	if err != nil {
		return false, errors.WithStack(err)
	}
	log.Logger.Info("Checking if an EKS cluster already exists...")
	log.Logger.Tracef("Checking if a Eks cluster config exists for values: %#v", templatedVars)

	args := []string{"get", "cluster"}
	args = parameteriseValues(args, p.eksConfig.Params.Global)
	args = parameteriseValues(args, p.eksConfig.Params.GetCluster)

	var stdoutBuf, stderrBuf bytes.Buffer

	err = utils.ExecCommand(p.eksConfig.Binary, args, map[string]string{}, &stdoutBuf,
		&stderrBuf, "", eksCommandTimeoutSeconds, 0, false)
	if err != nil {
		if errors.Cause(err) == context.DeadlineExceeded {
			return false, errors.Wrap(err,
				"Timed out trying to retrieve EKS cluster config. "+
					"Check your credentials.")
		}

		// todo - catch errors due to missing/expired AWS credentials and throw an error
		if _, ok := errors.Cause(err).(*exec.ExitError); ok {
			log.Logger.Info("EKS cluster doesn't exist")
			return false, nil
		} else {
			return false, errors.Wrap(err, "Error fetching EKS clusters")
		}
	}

	return true, nil
}

// Writes any configuration for a config file to a temporary file and returns it. If
// that key doesn't exist, an empty path is returned.
func (p EksProvisioner) writeConfigFile() (string, error) {
	if len(p.eksConfig.Params.ConfigFile) > 0 {

		// marshal the struct to YAML
		yamlBytes, err := yaml.Marshal(&p.eksConfig.Params.ConfigFile)
		if err != nil {
			return "", errors.WithStack(err)
		}

		yamlString := string(yamlBytes[:])

		// write the config to a temporary file
		tmpfile, err := ioutil.TempFile("", "eks.*.yaml")
		if err != nil {
			return "", errors.WithStack(err)
		}

		defer tmpfile.Close()

		if _, err := tmpfile.Write([]byte(yamlString)); err != nil {
			return "", errors.WithStack(err)
		}
		if err := tmpfile.Close(); err != nil {
			return "", errors.WithStack(err)
		}

		log.Logger.Debugf("EKS config file written to: %s", tmpfile.Name())

		return tmpfile.Name(), nil

	} else {
		log.Logger.Infof("No EKS config file data configured. No config file path will be passed " +
			"to eksctl commands")

		return "", nil
	}
}

// Creates an EKS cluster.
func (p EksProvisioner) Create(dryRun bool) error {

	clusterExists, err := p.clusterExists()
	if err != nil {
		return errors.WithStack(err)
	}

	if clusterExists {
		log.Logger.Debugf("An EKS cluster already exists called '%s'. Won't recreate it...",
			p.GetStack().GetConfig().GetCluster())
		return nil
	}

	templatedVars, err := p.stack.GetTemplatedVars(nil, map[string]interface{}{})
	if err != nil {
		return errors.WithStack(err)
	}
	log.Logger.Debugf("Templated stack config vars: %#v", templatedVars)

	args := []string{"create", "cluster"}
	args = parameteriseValues(args, p.eksConfig.Params.Global)
	args = parameteriseValues(args, p.eksConfig.Params.CreateCluster)

	configFilePath, err := p.writeConfigFile()
	if err != nil {
		return errors.WithStack(err)
	}

	if configFilePath != "" {
		args = append(args, []string{"-f", configFilePath}...)
	}

	var stdoutBuf, stderrBuf bytes.Buffer

	_, err = printer.Fprintf("Creating EKS cluster (this may take some time)...\n")
	if err != nil {
		return errors.WithStack(err)
	}

	err = utils.ExecCommand(p.eksConfig.Binary, args, map[string]string{}, &stdoutBuf,
		&stderrBuf, "", 0, 0, dryRun)
	if err != nil {
		return errors.WithStack(err)
	}

	if !dryRun {
		log.Logger.Debugf("EKS returned:\n%s", stdoutBuf.String())
		log.Logger.Infof("EKS cluster created")
	}

	p.stack.GetStatus().SetStartedThisRun(true)
	// only sleep before checking the cluster fo readiness if we started it
	p.stack.GetStatus().SetSleepBeforeReadyCheck(eksSleepSecondsBeforeReadyCheck)

	return nil
}

// Deletes a cluster
func (p EksProvisioner) Delete(approved bool, dryRun bool) error {
	dryRunPrefix := ""
	if dryRun {
		dryRunPrefix = "[Dry run] "
	}

	clusterExists, err := p.clusterExists()
	if err != nil {
		return errors.WithStack(err)
	}

	if !clusterExists {
		return errors.New("No EKS cluster exists to delete")
	}

	templatedVars, err := p.stack.GetTemplatedVars(nil, map[string]interface{}{})
	if err != nil {
		return errors.WithStack(err)
	}
	log.Logger.Debugf("Templated stack config vars: %#v", templatedVars)

	args := []string{"delete", "cluster"}
	args = parameteriseValues(args, p.eksConfig.Params.Global)
	args = parameteriseValues(args, p.eksConfig.Params.DeleteCluster)

	if approved {
		args = append(args, "--yes")
	}

	var stdoutBuf, stderrBuf bytes.Buffer

	if approved {
		_, err = printer.Fprintf("%sDeleting EKS cluster...\n", dryRunPrefix)
		if err != nil {
			return errors.WithStack(err)
		}
	} else {
		log.Logger.Infof("%sTesting deleting EKS cluster. Pass --yes to actually delete it", dryRunPrefix)
	}
	err = utils.ExecCommand(p.eksConfig.Binary, args, map[string]string{}, &stdoutBuf,
		&stderrBuf, "", eksCommandTimeoutSecondsLong, 0, dryRun)
	if err != nil {
		return errors.WithStack(err)
	}

	if approved {
		log.Logger.Infof("%sEKS cluster deleted...", dryRunPrefix)
	} else {
		log.Logger.Infof("%sEKS deletion test succeeded. Run with --yes to actually delete "+
			"the eks cluster", dryRunPrefix)
	}

	return nil
}

// Returns a boolean indicating whether the cluster is already online
func (p EksProvisioner) IsAlreadyOnline(dryRun bool) (bool, error) {

	// todo - test access by running kubectl get ns

	if dryRun {
		// say we'll check but don't actually check
		log.Logger.Debug("[Dry run] Checking whether a cluster config already exists")
		return true, nil
	}

	clusterExists, err := p.clusterExists()
	if err != nil {
		return false, errors.WithStack(err)
	}

	if !clusterExists {
		return false, nil
	}

	clusterSot := p.ClusterSot()
	online, err := clustersot.IsOnline(clusterSot)
	if err != nil {
		return false, errors.WithStack(err)
	}

	return online, nil
}

// Updates the cluster
func (p EksProvisioner) Update(dryRun bool) error {

	log.Logger.Infof("Updating EKS cluster '%s'...", p.eksConfig.clusterName)
	// todo make the --yes flag configurable, perhaps through a CLI arg so people can verify their
	// changes before applying them
	args := []string{
		"update",
		"cluster",
		"--yes",
	}

	args = parameteriseValues(args, p.eksConfig.Params.Global)
	args = parameteriseValues(args, p.eksConfig.Params.UpdateCluster)

	kubeConfig, _ := p.stack.GetRegistry().Get(constants.RegistryKeyKubeConfig)
	envVars := map[string]string{
		"KUBECONFIG": kubeConfig.(string),
	}

	var stdoutBuf, stderrBuf bytes.Buffer

	log.Logger.Info("Running eksctl update...")
	// this command might take a long time to complete so don't supply a timeout
	err := utils.ExecCommand(p.eksConfig.Binary, args, envVars, &stdoutBuf, &stderrBuf,
		"", 0, 0, dryRun)
	if err != nil {
		return errors.WithStack(err)
	}

	if !dryRun {
		log.Logger.Debugf("eksctl returned:\n%s", stdoutBuf.String())
		log.Logger.Infof("EKS cluster updated")
	}

	return nil
}

// Parses the Eks provisioner config
func parseEksConfig(stack interfaces.IStack) (*EksConfig, error) {
	templatedVars, err := stack.GetTemplatedVars(nil, map[string]interface{}{})
	if err != nil {
		return nil, errors.WithStack(err)
	}
	provisionerValues, ok := templatedVars[ProvisionerKey].(map[interface{}]interface{})
	if !ok {
		return nil, errors.New("No provisioner found in stack config. You must at least set the binary path.")
	}
	log.Logger.Tracef("Marshalling: %#v", provisionerValues)

	// marshal then unmarshal the provisioner values to get the command parameters
	byteData, err := yaml.Marshal(provisionerValues)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	log.Logger.Tracef("Marshalled to: %s", string(byteData[:]))

	var eksConfig EksConfig
	err = yaml.Unmarshal(byteData, &eksConfig)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	if eksConfig.Binary == "" {
		eksConfig.Binary = eksDefaultBinary
		log.Logger.Warnf("Using default %s binary '%s'. It's safer to explicitly set the path to a versioned "+
			"binary (e.g. %s-1.2.3) in the provisioner configuration", eksProvisionerName, eksDefaultBinary,
			eksDefaultBinary)
	}

	eksConfig.clusterName = eksConfig.Params.Global[configKeyEKSClusterName]

	return &eksConfig, nil
}

// No special connectivity is required for this provisioner
func (p *EksProvisioner) EnsureClusterConnectivity() (bool, error) {
	return true, nil
}

// Downloads the kubeconfig file for the cluster to a temporary location and
// returns the path to it
func (p EksProvisioner) downloadKubeConfigFile() (string, error) {

	log.Logger.Debugf("Downloading kubeconfig file for '%s'...",
		p.eksConfig.clusterName)

	pattern := fmt.Sprintf("kubeconfig-%s-*", p.GetStack().GetConfig().GetCluster())

	tmpfile, err := ioutil.TempFile("", pattern)
	if err != nil {
		return "", errors.WithStack(err)
	}

	kubeConfigPath := tmpfile.Name()

	var stdoutBuf, stderrBuf bytes.Buffer
	args := []string{"utils", "write-kubeconfig"}
	args = parameteriseValues(args, p.eksConfig.Params.Global)
	args = append(args, []string{"--kubeconfig", kubeConfigPath}...)

	err = utils.ExecCommand(p.eksConfig.Binary, args,
		nil, &stdoutBuf, &stderrBuf,
		"", eksSleepSecondsBeforeReadyCheck, 0, false)
	if err != nil {
		return "", errors.WithStack(err)
	}

	log.Logger.Infof("Kubeconfig file downloaded to '%s'", kubeConfigPath)

	return kubeConfigPath, nil
}

// Nothing to do for this provisioner
func (p EksProvisioner) Close() error {
	return nil
}
