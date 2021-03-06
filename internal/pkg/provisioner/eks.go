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
	"k8s.io/client-go/tools/clientcmd"
	"os"
	"os/exec"
	"regexp"
)

const eksProvisionerName = "eks"
const eksDefaultBinary = "eksctl"

const eksCommandTimeoutSeconds = 30
const eksCommandTimeoutSecondsLong = 300

// number of seconds to sleep after the cluster has come online before checking whether
// it's ready
const eksSleepSecondsBeforeReadyCheck = 15

// todo - catch errors accessing these
const configKeyEKSClusterName = "name"

type EksProvisioner struct {
	clusterSot           interfaces.IClusterSot
	stack                interfaces.IStack
	eksConfig            EksConfig
	portForwardingActive bool
	sshCommand           *exec.Cmd
}

type EksUtilsConfig struct {
	WriteKubeConfig map[string]string `yaml:"write_kubeconfig"`
}

type EksConfig struct {
	clusterName string // set after parsing the eks YAML
	Binary      string // path to the eksctl binary
	Params      struct {
		Global        map[string]string
		GetCluster    map[string]string `yaml:"get_cluster"`
		CreateCluster map[string]string `yaml:"create_cluster"`
		DeleteCluster map[string]string `yaml:"delete_cluster"`
		UpdateCluster map[string]string `yaml:"update_cluster"`
		Utils         EksUtilsConfig
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

func (p EksProvisioner) Binary() string {
	return p.eksConfig.Binary
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

	_, err = printer.Fprintf("Creating EKS cluster (this may take some time)...\n")
	if err != nil {
		return errors.WithStack(err)
	}

	err = utils.ExecCommandUnbuffered(p.eksConfig.Binary, args, map[string]string{},
		os.Stdout, os.Stderr, "", 0, 0, dryRun)
	if err != nil {
		return errors.WithStack(err)
	}

	if !dryRun {
		log.Logger.Infof("EKS cluster created")

		err = p.renameKubeContext()
		if err != nil {
			return errors.WithStack(err)
		}
	}

	p.stack.GetStatus().SetStartedThisRun(true)
	// only sleep before checking the cluster fo readiness if we started it
	p.stack.GetStatus().SetSleepBeforeReadyCheck(eksSleepSecondsBeforeReadyCheck)

	return nil
}

// When eksctl downloads a kubeconfig file for a cluster it uses the IAM username as the
// name of the kubecontext. This would complicate configuring the kubecontext, so let's
// just strip the username from the kubecontext
func (p EksProvisioner) renameKubeContext() error {
	log.Logger.Debugf("Renaming kube context for EKS cluster '%s'", p.eksConfig.clusterName)

	pathOptions := clientcmd.NewDefaultPathOptions()
	kubeConfig, err := pathOptions.GetStartingConfig()
	if err != nil {
		return errors.WithStack(err)
	}

	shortClusterName := p.eksConfig.clusterName

	clusterNameRe := regexp.MustCompile(fmt.Sprintf(".*@%s.%s.eksctl.io", shortClusterName,
		p.stack.GetConfig().GetRegion()))
	contextName := ""
	fullClusterName := ""

	for name, ctx := range kubeConfig.Contexts {
		if clusterNameRe.MatchString(name) {
			log.Logger.Debugf("Kubeconfig context '%s' matches regex '%s'", name, clusterNameRe.String())
			contextName = name
			fullClusterName = ctx.Cluster
		}
	}

	if contextName != "" {
		log.Logger.Infof("Renaming EKS cluster contex† from '%s' to '%s' in kubeconfig file",
			contextName, fullClusterName)

		kubeConfig.Contexts[fullClusterName] = kubeConfig.Contexts[contextName]
		delete(kubeConfig.Contexts, contextName)

		// also set the renamed cluster as the default context
		kubeConfig.CurrentContext = fullClusterName

		err = clientcmd.ModifyConfig(pathOptions, *kubeConfig, false)
		if err != nil {
			return errors.WithStack(err)
		}
	} else {
		log.Logger.Infof("Not renaming cluster context for EKS cluster '%s'", shortClusterName)
	}

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

	configFilePath, err := p.writeConfigFile()
	if err != nil {
		return errors.WithStack(err)
	}

	if configFilePath != "" {
		args = append(args, []string{"-f", configFilePath}...)
	}

	if approved {
		_, err = printer.Fprintf("%sDeleting EKS cluster...\n", dryRunPrefix)
		if err != nil {
			return errors.WithStack(err)
		}

		err = utils.ExecCommandUnbuffered(p.eksConfig.Binary, args, map[string]string{}, os.Stdout,
			os.Stderr, "", eksCommandTimeoutSecondsLong, 0, dryRun)
		if err != nil {
			return errors.WithStack(err)
		}
	} else {
		log.Logger.Infof("%sNo way to test deleting EKS clusters with eksctl. Pass --yes to actually delete it", dryRunPrefix)
	}

	if approved {
		log.Logger.Infof("%sEKS cluster deleted...", dryRunPrefix)
	} else {
		log.Logger.Infof("%sEKS cluster deletions cannot be tested. Run with --yes to actually delete "+
			"the cluster", dryRunPrefix)
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
	args := []string{
		"update",
		"cluster",
	}

	args = parameteriseValues(args, p.eksConfig.Params.Global)
	args = parameteriseValues(args, p.eksConfig.Params.UpdateCluster)

	configFilePath, err := p.writeConfigFile()
	if err != nil {
		return errors.WithStack(err)
	}

	if configFilePath != "" {
		args = append(args, []string{"-f", configFilePath}...)
	}

	kubeConfig, _ := p.stack.GetRegistry().Get(constants.RegistryKeyKubeConfig)
	envVars := map[string]string{
		constants.KubeConfigEnvVar: kubeConfig.(string),
	}

	log.Logger.Info("Running eksctl update...")
	// this command might take a long time to complete so don't supply a timeout
	err = utils.ExecCommandUnbuffered(p.eksConfig.Binary, args, envVars, os.Stdout,
		os.Stderr, "", 0, 0, dryRun)
	if err != nil {
		return errors.WithStack(err)
	}

	if !dryRun {
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

	eksConfig.clusterName = eksConfig.Params.GetCluster[configKeyEKSClusterName]

	return &eksConfig, nil
}

// Don't need to do anything here assuming the kubeconfig file was downloaded when
// the cluster was created
func (p *EksProvisioner) EnsureClusterConnectivity() (bool, error) {
	return true, nil
}

// Nothing to do for this provisioner
func (p EksProvisioner) Close() error {
	return nil
}
