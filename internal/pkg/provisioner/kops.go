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

package provisioner

import (
	"bytes"
	"context"
	"fmt"
	"github.com/imdario/mergo"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/sugarkube/sugarkube/internal/pkg/clustersot"
	"github.com/sugarkube/sugarkube/internal/pkg/constants"
	"github.com/sugarkube/sugarkube/internal/pkg/convert"
	"github.com/sugarkube/sugarkube/internal/pkg/interfaces"
	"github.com/sugarkube/sugarkube/internal/pkg/log"
	"github.com/sugarkube/sugarkube/internal/pkg/printer"
	"github.com/sugarkube/sugarkube/internal/pkg/sshtunnel"
	"github.com/sugarkube/sugarkube/internal/pkg/utils"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	log2 "log"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

const kopsProvisionerName = "kops"
const kopsDefaultBinary = "kops"

// number of seconds to timeout after while running kops commands (apart from updates)
const kopsCommandTimeoutSeconds = 30
const kopsCommandTimeoutSecondsLong = 300

// number of seconds to sleep after the cluster has come online before checking whether
// it's ready
const kopsSleepSecondsBeforeReadyCheck = 60

// todo - catch errors accessing each of these
const configKeyKopsSpecs = "specs"
const configKeyKopsCluster = "cluster"
const configKeyKopsInstanceGroups = "instanceGroups"
const configKeyApiLoadBalancerType = "api_loadbalancer_type"
const configKeyCreateCluster = "create_cluster"
const configKeyBastion = "bastion"
const configKeyClusterName = "name"

const awsCliPath = "aws"

const localhost = "127.0.0.1"
const etcHostsPath = "/etc/hosts"
const kubernetesLocalHostname = "kubernetes.default.svc.cluster.local"

type KopsProvisioner struct {
	clusterSot           interfaces.IClusterSot
	stack                interfaces.IStack
	kopsConfig           KopsConfig
	portForwardingActive bool
}

type KopsConfig struct {
	clusterName             string // set after parsing the kops YAML
	Binary                  string // path to the kops binary
	SshPrivateKey           string `yaml:"ssh_private_key"`
	BastionUser             string `yaml:"bastion_user"`
	LocalPortForwardingPort int    `yaml:"local_port_forwarding_port"`
	Params                  struct {
		Global            map[string]string
		CreateCluster     map[string]string `yaml:"create_cluster"`
		DeleteCluster     map[string]string `yaml:"delete_cluster"`
		UpdateCluster     map[string]string `yaml:"update_cluster"`
		GetClusters       map[string]string `yaml:"get_clusters"`
		GetInstanceGroups map[string]string `yaml:"get_instance_groups"`
		RollingUpdate     map[string]string `yaml:"rolling_update"`
		Replace           map[string]string
	}
}

// Instantiates a new instance
func newKopsProvisioner(stackConfig interfaces.IStack, clusterSot interfaces.IClusterSot) (*KopsProvisioner, error) {
	kopsConfig, err := parseKopsConfig(stackConfig)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return &KopsProvisioner{
		stack:      stackConfig,
		kopsConfig: *kopsConfig,
		clusterSot: clusterSot,
	}, nil
}

func (p KopsProvisioner) GetStack() interfaces.IStack {
	return p.stack
}

func (p KopsProvisioner) ClusterSot() interfaces.IClusterSot {
	return p.clusterSot
}

// Returns a bool indicating whether the cluster configuration has already been created
func (p KopsProvisioner) clusterConfigExists() (bool, error) {

	templatedVars, err := p.stack.GetTemplatedVars(nil, map[string]interface{}{})
	if err != nil {
		return false, errors.WithStack(err)
	}
	log.Logger.Info("Checking if a Kops cluster config already exists...")
	log.Logger.Tracef("Checking if a Kops cluster config exists for values: %#v", templatedVars)

	args := []string{"get", "clusters"}
	args = parameteriseValues(args, p.kopsConfig.Params.Global)
	args = parameteriseValues(args, p.kopsConfig.Params.GetClusters)

	var stdoutBuf, stderrBuf bytes.Buffer

	err = utils.ExecCommand(p.kopsConfig.Binary, args, map[string]string{}, &stdoutBuf,
		&stderrBuf, "", kopsCommandTimeoutSeconds, 0, false)
	if err != nil {
		if errors.Cause(err) == context.DeadlineExceeded {
			return false, errors.Wrap(err,
				"Timed out trying to retrieve kops cluster config. "+
					"Check your credentials.")
		}

		// todo - catch errors due to missing/expired AWS credentials and throw an error
		if _, ok := errors.Cause(err).(*exec.ExitError); ok {
			log.Logger.Info("Kops cluster config doesn't exist")
			return false, nil
		} else {
			return false, errors.Wrap(err, "Error fetching kops clusters")
		}
	}

	return true, nil
}

// Creates a Kops cluster config. Note: This doesn't actually launch a Kops cluster,
// that only happens when 'kops update' is run.
func (p KopsProvisioner) Create(dryRun bool) error {

	configExists, err := p.clusterConfigExists()
	if err != nil {
		return errors.WithStack(err)
	}

	if configExists {
		log.Logger.Debugf("Kops config already exists for '%s'. Won't recreate it...",
			p.GetStack().GetConfig().GetCluster())
		return nil
	}

	templatedVars, err := p.stack.GetTemplatedVars(nil, map[string]interface{}{})
	if err != nil {
		return errors.WithStack(err)
	}
	log.Logger.Debugf("Templated stack config vars: %#v", templatedVars)

	args := []string{"create", "cluster"}
	args = parameteriseValues(args, p.kopsConfig.Params.Global)
	args = parameteriseValues(args, p.kopsConfig.Params.CreateCluster)

	var stdoutBuf, stderrBuf bytes.Buffer

	_, err = printer.Fprintf("Creating kops cluster config...\n")
	if err != nil {
		return errors.WithStack(err)
	}

	err = utils.ExecCommand(p.kopsConfig.Binary, args, map[string]string{}, &stdoutBuf,
		&stderrBuf, "", kopsCommandTimeoutSecondsLong, 0, dryRun)
	if err != nil {
		return errors.WithStack(err)
	}

	if !dryRun {
		log.Logger.Debugf("Kops returned:\n%s", stdoutBuf.String())
		log.Logger.Infof("Kops cluster config created")
	}

	err = p.patch(dryRun)
	if err != nil {
		return errors.WithStack(err)
	}

	p.stack.GetStatus().SetStartedThisRun(true)
	// only sleep before checking the cluster fo readiness if we started it
	p.stack.GetStatus().SetSleepBeforeReadyCheck(kopsSleepSecondsBeforeReadyCheck)

	return nil
}

// Deletes a cluster
func (p KopsProvisioner) Delete(approved bool, dryRun bool) error {
	dryRunPrefix := ""
	if dryRun {
		dryRunPrefix = "[Dry run] "
	}

	configExists, err := p.clusterConfigExists()
	if err != nil {
		return errors.WithStack(err)
	}

	if !configExists {
		return errors.New("No kops cluster config exists to delete")
	}

	templatedVars, err := p.stack.GetTemplatedVars(nil, map[string]interface{}{})
	if err != nil {
		return errors.WithStack(err)
	}
	log.Logger.Debugf("Templated stack config vars: %#v", templatedVars)

	args := []string{"delete", "cluster"}
	args = parameteriseValues(args, p.kopsConfig.Params.Global)
	args = parameteriseValues(args, p.kopsConfig.Params.DeleteCluster)

	if approved {
		args = append(args, "--yes")
	}

	var stdoutBuf, stderrBuf bytes.Buffer

	if approved {
		_, err = printer.Fprintf("%sDeleting kops cluster...\n", dryRunPrefix)
		if err != nil {
			return errors.WithStack(err)
		}
	} else {
		log.Logger.Infof("%sTesting deleting Kops cluster. Pass --yes to actually delete it", dryRunPrefix)
	}
	err = utils.ExecCommand(p.kopsConfig.Binary, args, map[string]string{}, &stdoutBuf,
		&stderrBuf, "", kopsCommandTimeoutSecondsLong, 0, dryRun)
	if err != nil {
		return errors.WithStack(err)
	}

	if approved {
		log.Logger.Infof("%sKops cluster deleted...", dryRunPrefix)
	} else {
		log.Logger.Infof("%sKops deletion test succeeded. Run with --yes to actually delete "+
			"the kops cluster", dryRunPrefix)
	}

	return nil
}

// Returns a boolean indicating whether the cluster is already online
func (p KopsProvisioner) IsAlreadyOnline(dryRun bool) (bool, error) {

	if dryRun {
		// say we'll check but don't actually check
		log.Logger.Debug("[Dry run] Checking whether a cluster config already exists")
		return true, nil
	}

	configExists, err := p.clusterConfigExists()
	if err != nil {
		return false, errors.WithStack(err)
	}

	if !configExists {
		return false, nil
	}

	clusterSot := p.ClusterSot()
	online, err := clustersot.IsOnline(clusterSot)
	if err != nil {
		return false, errors.WithStack(err)
	}

	return online, nil
}

// No-op function, required to fully implement the Provisioner interface
func (p KopsProvisioner) Update(dryRun bool) error {

	err := p.patch(dryRun)
	if err != nil {
		return errors.WithStack(err)
	}

	log.Logger.Infof("Performing a rolling update to apply config changes to the kops cluster...")
	// todo make the --yes flag configurable, perhaps through a CLI arg so people can verify their
	// changes before applying them
	args := []string{
		"rolling-update",
		"cluster",
		"--yes",
	}

	args = parameteriseValues(args, p.kopsConfig.Params.Global)
	args = parameteriseValues(args, p.kopsConfig.Params.RollingUpdate)

	kubeConfig, _ := p.stack.GetRegistry().Get(constants.RegistryKeyKubeConfig)
	envVars := map[string]string{
		"KUBECONFIG": kubeConfig.(string),
	}

	var stdoutBuf, stderrBuf bytes.Buffer

	log.Logger.Info("Running Kops rolling update...")
	// this command might take a long time to complete so don't supply a timeout
	err = utils.ExecCommand(p.kopsConfig.Binary, args, envVars, &stdoutBuf, &stderrBuf,
		"", 0, 0, dryRun)
	if err != nil {
		return errors.WithStack(err)
	}

	if !dryRun {
		log.Logger.Debugf("Kops returned:\n%s", stdoutBuf.String())
		log.Logger.Infof("Kops cluster updated")
	}

	return nil
}

// Patches a Kops cluster configuration. Downloads the current config then merges in any configured
// spec.
func (p KopsProvisioner) patch(dryRun bool) error {
	var err error

	if dryRun {
		// say we'll check but don't actually check
		log.Logger.Debug("[Dry run] Checking whether a cluster config already exists")
	} else {
		configExists, err := p.clusterConfigExists()
		if err != nil {
			return errors.WithStack(err)
		}

		// can't update a non-existent config
		if !configExists {
			return nil
		}
	}

	// get the kops config
	args := []string{
		"get",
		"clusters",
		"-o",
		"yaml",
	}

	args = parameteriseValues(args, p.kopsConfig.Params.Global)
	args = parameteriseValues(args, p.kopsConfig.Params.GetClusters)

	var stdoutBuf, stderrBuf bytes.Buffer

	log.Logger.Info("Downloading config for kops cluster...")

	err = utils.ExecCommand(p.kopsConfig.Binary, args, map[string]string{}, &stdoutBuf, &stderrBuf,
		"", kopsCommandTimeoutSeconds, 0, dryRun)
	if err != nil {
		return errors.WithStack(err)
	}

	if !dryRun {
		log.Logger.Tracef("Downloaded config for kops cluster:\n%s", stdoutBuf.String())
	}

	kopsYamlConfig := map[string]interface{}{}
	err = yaml.Unmarshal(stdoutBuf.Bytes(), kopsYamlConfig)
	if err != nil {
		return errors.Wrap(err, "Error parsing kops config")
	}

	templatedVars, err := p.stack.GetTemplatedVars(nil, map[string]interface{}{})
	if err != nil {
		return errors.WithStack(err)
	}
	provisionerValues := templatedVars[ProvisionerKey].(map[interface{}]interface{})

	specs, err := convert.MapInterfaceInterfaceToMapStringInterface(
		provisionerValues[configKeyKopsSpecs].(map[interface{}]interface{}))
	if err != nil {
		return errors.WithStack(err)
	}

	clusterSpecs := specs[configKeyKopsCluster]

	specValues := map[string]interface{}{"spec": clusterSpecs}

	log.Logger.Tracef("Spec to merge in:\n%s", specValues)

	// patch in the configured spec
	err = mergo.Merge(&kopsYamlConfig, specValues, mergo.WithOverride)
	if err != nil {
		return errors.WithStack(err)
	}

	log.Logger.Tracef("Merged config is:\n%s", kopsYamlConfig)

	yamlBytes, err := yaml.Marshal(&kopsYamlConfig)
	if err != nil {
		return errors.WithStack(err)
	}

	yamlString := string(yamlBytes[:])
	log.Logger.Tracef("Merged config:\n%s", yamlString)

	// todo - if the merged values are the same as the original, skip replacing the config

	// write the merged data to a temp file because we can't pipe it into kops
	tmpfile, err := ioutil.TempFile("", "kops.*.yaml")
	if err != nil {
		return errors.WithStack(err)
	}

	defer tmpfile.Close()

	if _, err := tmpfile.Write([]byte(yamlString)); err != nil {
		return errors.WithStack(err)
	}
	if err := tmpfile.Close(); err != nil {
		return errors.WithStack(err)
	}

	_, err = printer.Fprintf("Replacing kops cluster config...\n")
	if err != nil {
		return errors.WithStack(err)
	}

	// Replace the cluster config
	args = []string{
		"replace",
		"-f",
		tmpfile.Name(),
	}

	args = parameteriseValues(args, p.kopsConfig.Params.Global)
	args = parameteriseValues(args, p.kopsConfig.Params.Replace)

	err = utils.ExecCommand(p.kopsConfig.Binary, args, map[string]string{}, &stdoutBuf, &stderrBuf,
		"", kopsCommandTimeoutSeconds, 0, dryRun)
	if err != nil {
		return errors.WithStack(err)
	}

	// we don't just defer this because it's useful to inspect it if the above command fails
	err = os.Remove(tmpfile.Name()) // clean up
	if err != nil {
		return errors.WithStack(err)
	}

	if !dryRun {
		log.Logger.Info("Kops cluster config replaced.")
	}

	log.Logger.Info("Patching instance group configs...")

	igSpecs, ok := specs[configKeyKopsInstanceGroups]
	if ok {
		for instanceGroupName, newSpec := range igSpecs.(map[interface{}]interface{}) {
			specValues := map[string]interface{}{"spec": newSpec}
			err = p.patchInstanceGroup(p.kopsConfig, instanceGroupName.(string), specValues, dryRun)
			if err != nil {
				return errors.WithStack(err)
			}
		}
	}

	args = []string{
		"update",
		"cluster",
		"--yes",
	}

	args = parameteriseValues(args, p.kopsConfig.Params.Global)
	args = parameteriseValues(args, p.kopsConfig.Params.UpdateCluster)

	_, err = printer.Fprintf("Updating kops cluster...\n")
	if err != nil {
		return errors.WithStack(err)
	}

	// this command might take a long time to complete so don't supply a timeout
	err = utils.ExecCommand(p.kopsConfig.Binary, args, map[string]string{}, &stdoutBuf, &stderrBuf,
		"", 0, 0, dryRun)
	if err != nil {
		return errors.WithStack(err)
	}

	return nil
}

func (p KopsProvisioner) patchInstanceGroup(kopsConfig KopsConfig, instanceGroupName string,
	newSpec map[string]interface{}, dryRun bool) error {
	args := []string{
		"get",
		"instancegroups",
		instanceGroupName,
		"-o",
		"yaml",
	}

	args = parameteriseValues(args, kopsConfig.Params.Global)
	args = parameteriseValues(args, kopsConfig.Params.GetInstanceGroups)

	log.Logger.Infof("Downloading config for instance group '%s' from kops cluster", instanceGroupName)

	var stdoutBuf, stderrBuf bytes.Buffer
	err := utils.ExecCommand(kopsConfig.Binary, args, map[string]string{}, &stdoutBuf,
		&stderrBuf, "", kopsCommandTimeoutSeconds, 0, dryRun)
	if err != nil {
		return errors.WithStack(err)
	}

	log.Logger.Tracef("Downloaded instance group config is:\n%s",
		stdoutBuf.String())

	kopsYamlConfig := map[string]interface{}{}
	err = yaml.Unmarshal(stdoutBuf.Bytes(), kopsYamlConfig)
	if err != nil {
		return errors.Wrap(err, "Error parsing kops instance group config")
	}
	log.Logger.Tracef("Yaml instance group kopsYamlConfig:\n%s", kopsYamlConfig)

	// patch in the configured spec
	err = mergo.Merge(&kopsYamlConfig, newSpec, mergo.WithOverride)
	if err != nil {
		return errors.WithStack(err)
	}

	log.Logger.Tracef("Merged instance group config is:\n%s", kopsYamlConfig)

	yamlBytes, err := yaml.Marshal(&kopsYamlConfig)
	if err != nil {
		return errors.WithStack(err)
	}

	yamlString := string(yamlBytes[:])
	log.Logger.Tracef("Merged config:\n%s", yamlString)

	// write the merged data to a temp file because we can't pipe it into kops
	tmpfile, err := ioutil.TempFile("", "kops.*.yaml")
	if err != nil {
		return errors.WithStack(err)
	}

	defer tmpfile.Close()

	if _, err := tmpfile.Write([]byte(yamlString)); err != nil {
		return errors.WithStack(err)
	}
	if err := tmpfile.Close(); err != nil {
		return errors.WithStack(err)
	}

	// replace the cluster config

	log.Logger.Infof("Replacing config of Kops instance group %s...", instanceGroupName)
	args = []string{
		"replace",
		"-f",
		tmpfile.Name(),
	}

	args = parameteriseValues(args, kopsConfig.Params.Global)
	args = parameteriseValues(args, kopsConfig.Params.Replace)

	err = utils.ExecCommand(kopsConfig.Binary, args, map[string]string{}, &stdoutBuf, &stderrBuf,
		"", kopsCommandTimeoutSeconds, 0, dryRun)
	if err != nil {
		return errors.WithStack(err)
	}

	// we don't just defer this because it's useful to inspect it if the above command fails
	err = os.Remove(tmpfile.Name()) // clean up
	if err != nil {
		return errors.WithStack(err)
	}

	log.Logger.Infof("Successfully replaced config of instance group '%s'", instanceGroupName)

	return nil
}

// Converts an array of key-value parameters to CLI args
func parameteriseValues(args []string, valueMap map[string]string) []string {
	// In kops, booleans can only indicate truth. Passing e.g. `--bastion false`
	// trips it up. So we need to explicitly filter out such keys if they have
	// the corresponding values
	excludeBooleans := map[string]string{
		"bastion": "false",
	}

	for k, v := range valueMap {
		ignoreValue := false

		for excludeK, excludeV := range excludeBooleans {
			if k == excludeK && v == excludeV {
				log.Logger.Tracef("Ignoring kops parameter '%s' (which is '%v')",
					excludeK, excludeV)
				ignoreValue = true
			}
		}

		if ignoreValue {
			continue
		}

		key := strings.Replace(k, "_", "-", -1)

		value := fmt.Sprintf("%v", v)
		if value != "" {
			value = fmt.Sprintf("%v", v)
			// we need to separate keys & values with equals signs so kops doesn't
			// get tripped up with `--bastion true`
			args = append(args, fmt.Sprintf("--%s=%s", key, value))
		} else {
			args = append(args, "--"+key)
		}
	}

	return args
}

// Parses the Kops provisioner config
func parseKopsConfig(stack interfaces.IStack) (*KopsConfig, error) {
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

	var kopsConfig KopsConfig
	err = yaml.Unmarshal(byteData, &kopsConfig)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	if kopsConfig.Binary == "" {
		kopsConfig.Binary = kopsDefaultBinary
		log.Logger.Warnf("Using default %s binary '%s'. It's safer to explicitly set the path to a versioned "+
			"binary (e.g. %s-1.2.3) in the provisioner configuration", kopsProvisionerName, kopsDefaultBinary,
			kopsDefaultBinary)
	}

	kopsConfig.clusterName = kopsConfig.Params.Global[configKeyClusterName]

	return &kopsConfig, nil
}

// Return a boolean indicating whether we need to set up port forwarding to the
// API server via the bastion
func (p KopsProvisioner) needApiAccessViaBastion() bool {

	// if the API load balancer type is internal and there's a bastion, we need
	// to set up port forwarding
	apiLoadBalancerType, ok := p.kopsConfig.Params.CreateCluster[configKeyApiLoadBalancerType]
	if !ok {
		log.Logger.Infof("No `%s` key under the kops `%s` key. Assuming we "+
			"don't need to set up SSH port forwarding to access the API server",
			configKeyApiLoadBalancerType, configKeyCreateCluster)
		return false
	}

	apiLoadBalancerType = strings.ToLower(apiLoadBalancerType)

	bastionValue, ok := p.kopsConfig.Params.CreateCluster[configKeyBastion]
	if !ok {
		log.Logger.Infof("No `%s` key under the kops `%s` key. Won't set up "+
			"SSH port forwarding to access the API server", configKeyBastion,
			configKeyCreateCluster)
	}

	bastionValue = strings.ToLower(bastionValue)

	return apiLoadBalancerType == "internal" && bastionValue == "" || bastionValue == "true"
}

// If the API server is internal and there's a bastion, set up SSH port forwarding
func (p *KopsProvisioner) EnsureClusterConnectivity() (bool, error) {

	if !p.needApiAccessViaBastion() {
		log.Logger.Debug("No need to set up SSH port forwarding to access " +
			"the API server")
		// no need to do anything
		return true, nil
	}

	if p.portForwardingActive {
		log.Logger.Debug("Port forwarding already set up")
		return true, nil
	}

	configExists, err := p.clusterConfigExists()
	if err != nil {
		return false, errors.WithStack(err)
	}

	if !configExists {
		log.Logger.Debug("Can't establish connectivity for non-existent kops cluster")
		return false, nil
	}

	_, err = printer.Fprintf("Setting up SSH port forwarding via the bastion to " +
		"the internal API server...\n")
	if err != nil {
		return false, errors.WithStack(err)
	}

	config := p.stack.GetConfig()

	bastionHostname, err := getBastionHostname(config)
	if err != nil {
		return false, errors.WithStack(err)
	}

	if bastionHostname == "" {
		log.Logger.Info("Won't set up port forwarding - no bastion found")
		return false, nil
	}

	err = assertInHostsFile(localhost, kubernetesLocalHostname)
	if err != nil {
		return false, errors.WithStack(err)
	}

	apiDomain := fmt.Sprintf("api.%s", p.kopsConfig.clusterName)

	var kubeConfigPathStr string
	kubeConfigPathInterface, _ := p.stack.GetRegistry().Get(constants.RegistryKeyKubeConfig)

	localSSHPort := p.setupPortForwarding(p.kopsConfig.SshPrivateKey, p.kopsConfig.BastionUser,
		bastionHostname, apiDomain, 443)

	p.kopsConfig.LocalPortForwardingPort = localSSHPort

	p.portForwardingActive = true

	if kubeConfigPathInterface != "" {
		kubeConfigPathStr = kubeConfigPathInterface.(string)
		log.Logger.Debugf("Kubeconfig file already downloaded to '%s'", kubeConfigPathStr)
		if _, err := os.Stat(kubeConfigPathStr); err != nil {
			log.Logger.Errorf("Kubeconfig file '%s' doesn't exist", kubeConfigPathStr)
			return false, fmt.Errorf("Kubeconfig file '%s' doesn't exist! If you have a "+
				"KUBECONFIG environment variable set, delete it and try again.", kubeConfigPathStr)
		}
	} else {
		kubeConfigPathStr, err = p.downloadKubeConfigFile()
		if err != nil {
			return false, errors.WithStack(err)
		}

		err := p.stack.GetRegistry().Set(constants.RegistryKeyKubeConfig, kubeConfigPathStr)
		if err != nil {
			return false, errors.WithStack(err)
		}

		// modify the host name in the file to point to the local k8s server domain
		err = replaceAllInFile(apiDomain, fmt.Sprintf("%s:%d", kubernetesLocalHostname, localSSHPort),
			kubeConfigPathStr)
		if err != nil {
			return false, errors.WithStack(err)
		}
	}

	_, err = printer.Fprintf("[green]SSH port forwarding established. Use [bold]KUBECONFIG=%s[reset]\n\n",
		kubeConfigPathStr)
	if err != nil {
		return true, errors.WithStack(err)
	}

	return true, nil
}

// Downloads the KUBECONFIG file for the cluster to a temporary location and
// returns the path to it
func (p KopsProvisioner) downloadKubeConfigFile() (string, error) {

	log.Logger.Debugf("Downloading kubeconfig file for '%s'...",
		p.kopsConfig.clusterName)

	pattern := fmt.Sprintf("kubeconfig-%s-*", p.GetStack().GetConfig().GetCluster())

	tmpfile, err := ioutil.TempFile("", pattern)
	if err != nil {
		return "", errors.WithStack(err)
	}

	kubeConfigPath := tmpfile.Name()

	var stdoutBuf, stderrBuf bytes.Buffer
	args := []string{"export", "kubecfg"}
	args = parameteriseValues(args, p.kopsConfig.Params.Global)

	err = utils.ExecCommand(p.kopsConfig.Binary, args,
		map[string]string{"KUBECONFIG": kubeConfigPath}, &stdoutBuf, &stderrBuf,
		"", kopsSleepSecondsBeforeReadyCheck, 0, false)
	if err != nil {
		return "", errors.WithStack(err)
	}

	log.Logger.Infof("Kubeconfig file downloaded to '%s'", kubeConfigPath)

	return kubeConfigPath, nil
}

// Replace all occurrences of a string in a file
func replaceAllInFile(search string, replacement string, path string) error {
	contents, err := ioutil.ReadFile(path)
	if err != nil {
		return errors.WithStack(err)
	}

	updated := strings.Replace(string(contents), search, replacement, -1)

	err = ioutil.WriteFile(path, []byte(updated), 0)
	if err != nil {
		return errors.WithStack(err)
	}

	return nil
}

// Sets up SSH port forwarding
func (p *KopsProvisioner) setupPortForwarding(privateKey string, sshUser string, sshHost string,
	remoteAddress string, remotePort int) int {

	privateKey = utils.ExpandUser(privateKey)

	// the bastion
	intermediateUserHost := strings.Join([]string{sshUser, sshHost}, "@")
	// the API server only accessible via the bastion
	remoteUserHost := fmt.Sprintf("%s:%d", remoteAddress, remotePort)

	log.Logger.Infof("Setting up SSH port forwarding for '%s' to '%s' with private SSH key '%s'",
		remoteUserHost, intermediateUserHost, privateKey)

	// Setup the tunnel, but do not yet start it yet.
	tunnel := sshtunnel.NewSSHTunnel(
		intermediateUserHost,
		sshtunnel.PrivateKeyFile(privateKey),
		remoteUserHost,
	)

	// configure logging for SSH tunnel if logging is enabled at certain levels
	if log.Logger.Level == logrus.TraceLevel || log.Logger.Level == logrus.DebugLevel {
		tunnel.Log = log2.New(os.Stderr, "sshtunnel ", log2.Ldate|log2.Lmicroseconds)
	}

	// maximum amount of time for the TCP connection to establish
	tunnel.Config.Timeout = time.Duration(5 * time.Second)

	go func() {
		// retry up to a certain number of times
		var err error
		for i := 5; i >= 0; i-- {
			err = tunnel.Start()
			if err != nil {
				log.Logger.Warnf("Error creating local SSH server for port forwarding: %s", err)
				log.Logger.Infof("Sleeping before trying again (%d tries left)", i)
				// sleep and retry
				time.Sleep(1 * time.Second)
			}
		}
	}()

	// sleep a little to bind to the local port
	time.Sleep(500 * time.Millisecond)
	log.Logger.Infof("Port forwarding server listening on local port: %d", tunnel.Local.Port)

	return tunnel.Local.Port
}

// Returns the hostname of the bastion or an empty string if it can't be found
func getBastionHostname(stackConfig interfaces.IStackConfig) (string, error) {
	var stdoutBuf, stderrBuf bytes.Buffer

	query := fmt.Sprintf("LoadBalancerDescriptions"+
		"[?contains(DNSName, `-%s-`)] | "+
		"[?contains(DNSName, `bastion`)].DNSName | [0]", stackConfig.GetCluster())

	// get the bastion ELB's hostname
	err := utils.ExecCommand(awsCliPath, []string{
		"--region", stackConfig.GetRegion(),
		"elb", "describe-load-balancers",
		"--query", query,
		"--output", "text",
	}, map[string]string{}, &stdoutBuf, &stderrBuf, "",
		kopsCommandTimeoutSeconds, 0, false)
	if err != nil {
		return "", errors.WithStack(err)
	}

	bastionHostname := strings.TrimSpace(stdoutBuf.String())

	if strings.ToLower(bastionHostname) == "none" {
		bastionHostname = ""
	}

	if bastionHostname == "" {
		log.Logger.Infof("No bastion found for cluster '%s'", stackConfig.GetCluster())
	} else {
		log.Logger.Infof("The bastion hostname is '%s'", bastionHostname)
	}
	return bastionHostname, nil
}

// Throws an error if an IP and domain aren't in /etc/hosts
func assertInHostsFile(ip string, domain string) error {

	contents, err := ioutil.ReadFile(etcHostsPath)
	if err != nil {
		return errors.WithStack(err)
	}

	contentsString := string(contents)

	match, err := regexp.MatchString(fmt.Sprintf("%s.*%s", ip, domain),
		contentsString)
	if err != nil {
		return errors.WithStack(err)
	}

	if !match {
		return errors.New(fmt.Sprintf("No entry for '%s %s' in %s",
			ip, domain, etcHostsPath))
	}

	return nil
}

// Delete the downloaded kubeconfig file if we set up and ssh tunnel
func (p KopsProvisioner) Close() error {
	if p.portForwardingActive {
		// delete the downloaded kubeconfig file
		kubeConfigPathInterface, _ := p.stack.GetRegistry().Get(constants.RegistryKeyKubeConfig)
		kubeConfigPath := kubeConfigPathInterface.(string)
		if _, err := os.Stat(kubeConfigPath); err == nil {
			// todo - make this configurable based on a CLI flag
			log.Logger.Infof("Deleting downloaded kubeconfig file from %s", kubeConfigPath)
			err = os.Remove(kubeConfigPath)
			if err != nil {
				return errors.WithStack(err)
			}
		}
	} else {
		log.Logger.Debug("SSH port forwarding wasn't set up so no need to shut it down.")
	}

	return nil
}
