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
	"github.com/sugarkube/sugarkube/internal/pkg/clustersot"
	"github.com/sugarkube/sugarkube/internal/pkg/convert"
	"github.com/sugarkube/sugarkube/internal/pkg/interfaces"
	"github.com/sugarkube/sugarkube/internal/pkg/kapp"
	"github.com/sugarkube/sugarkube/internal/pkg/log"
	"github.com/sugarkube/sugarkube/internal/pkg/utils"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
	"os/exec"
	"regexp"
	"strconv"
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
const sshPortForwardingDelaySeconds = 5

const configKeyKopsSpecs = "specs"
const configKeyKopsCluster = "cluster"
const configKeyKopsInstanceGroups = "instanceGroups"
const configKeyApiLoadBalancerType = "api_loadbalancer_type"
const configKeyCreateCluster = "create_cluster"
const configKeyBastion = "bastion"
const configKeyClusterName = "name"

const registryKeyKubeConfig = "kube_config"

const awsCliPath = "aws"
const sshPath = "ssh"

const defaultLocalPortForwardingPort = 8443
const localhost = "127.0.0.1"
const etcHostsPath = "/etc/hosts"
const kubernetesLocalHostname = "kubernetes.default.svc.cluster.local"

type KopsProvisioner struct {
	clusterSot           clustersot.ClusterSot
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
		UpdateCluster     map[string]string `yaml:"update_cluster"`
		GetClusters       map[string]string `yaml:"get_clusters"`
		GetInstanceGroups map[string]string `yaml:"get_instance_groups"`
		RollingUpdate     map[string]string `yaml:"rolling_update"`
		Replace           map[string]string
	}
}

// Instantiates a new instance
func newKopsProvisioner(stackConfig interfaces.IStack) (*KopsProvisioner, error) {
	kopsConfig, err := parseKopsConfig(stackConfig)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return &KopsProvisioner{
		stack:      stackConfig,
		kopsConfig: *kopsConfig,
	}, nil
}

func (p KopsProvisioner) iStack() interfaces.IStack {
	return p.stack
}

func (p KopsProvisioner) ClusterSot() (clustersot.ClusterSot, error) {
	if p.clusterSot == nil {
		clusterSot, err := clustersot.NewClusterSot(clustersot.KUBECTL, p.stack)
		if err != nil {
			return nil, errors.WithStack(err)
		}

		p.clusterSot = clusterSot
	}

	return p.clusterSot, nil
}

// Returns a bool indicating whether the cluster configuration has already been created
func (p KopsProvisioner) clusterConfigExists() (bool, error) {

	templatedVars, err := p.stack.TemplatedVars(nil, map[string]interface{}{})
	if err != nil {
		return false, errors.WithStack(err)
	}
	log.Logger.Info("Checking if a Kops cluster config already exists...")
	log.Logger.Debugf("Checking if a Kops cluster config exists for values: %#v", templatedVars)

	args := []string{"get", "clusters"}
	args = parameteriseValues(args, p.kopsConfig.Params.Global)
	args = parameteriseValues(args, p.kopsConfig.Params.GetClusters)

	var stdoutBuf, stderrBuf bytes.Buffer

	err = utils.ExecCommand(p.kopsConfig.Binary, args, map[string]string{}, &stdoutBuf,
		&stderrBuf, "", kopsCommandTimeoutSeconds, false)
	if err != nil {
		if errors.Cause(err) == context.DeadlineExceeded {
			return false, errors.Wrap(err,
				"Timed out trying to retrieve kops cluster config. "+
					"Check your credentials.")
		}

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
func (p KopsProvisioner) create(dryRun bool) error {

	configExists, err := p.clusterConfigExists()
	if err != nil {
		return errors.WithStack(err)
	}

	if configExists {
		log.Logger.Debugf("Kops config already exists for '%s'. Won't recreate it...",
			p.iStack().GetConfig().Cluster)
		return nil
	}

	templatedVars, err := p.stack.TemplatedVars(nil, map[string]interface{}{})
	if err != nil {
		return errors.WithStack(err)
	}
	log.Logger.Debugf("Templated stack config vars: %#v", templatedVars)

	args := []string{"create", "cluster"}
	args = parameteriseValues(args, p.kopsConfig.Params.Global)
	args = parameteriseValues(args, p.kopsConfig.Params.CreateCluster)

	var stdoutBuf, stderrBuf bytes.Buffer

	log.Logger.Info("Creating Kops cluster config...")
	err = utils.ExecCommand(p.kopsConfig.Binary, args, map[string]string{}, &stdoutBuf,
		&stderrBuf, "", kopsCommandTimeoutSecondsLong, dryRun)
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

// Returns a boolean indicating whether the cluster is already online
func (p KopsProvisioner) isAlreadyOnline() (bool, error) {
	configExists, err := p.clusterConfigExists()
	if err != nil {
		return false, errors.WithStack(err)
	}

	if !configExists {
		return false, nil
	}

	clusterSot, err := p.ClusterSot()
	if err != nil {
		return false, errors.WithStack(err)
	}

	online, err := clustersot.IsOnline(clusterSot)
	if err != nil {
		return false, errors.WithStack(err)
	}

	return online, nil
}

// No-op function, required to fully implement the Provisioner interface
func (p KopsProvisioner) update(dryRun bool) error {

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

	var stdoutBuf, stderrBuf bytes.Buffer

	log.Logger.Info("Running Kops rolling update...")
	err = utils.ExecCommand(p.kopsConfig.Binary, args, map[string]string{}, &stdoutBuf, &stderrBuf,
		"", 0, dryRun)
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
	configExists, err := p.clusterConfigExists()
	if err != nil {
		return errors.WithStack(err)
	}

	// can't update a non-existent config
	if !configExists {
		return nil
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
		"", kopsCommandTimeoutSeconds, dryRun)
	if err != nil {
		return errors.WithStack(err)
	}

	if !dryRun {
		log.Logger.Debugf("Downloaded config for kops cluster:\n%s", stdoutBuf.String())
	}

	kopsYamlConfig := map[string]interface{}{}
	err = yaml.Unmarshal(stdoutBuf.Bytes(), kopsYamlConfig)
	if err != nil {
		return errors.Wrap(err, "Error parsing kops config")
	}

	templatedVars, err := p.stack.TemplatedVars(nil, map[string]interface{}{})
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

	log.Logger.Debugf("Spec to merge in:\n%s", specValues)

	// patch in the configured spec
	err = mergo.Merge(&kopsYamlConfig, specValues, mergo.WithOverride)
	if err != nil {
		return errors.WithStack(err)
	}

	log.Logger.Debugf("Merged config is:\n%s", kopsYamlConfig)

	yamlBytes, err := yaml.Marshal(&kopsYamlConfig)
	if err != nil {
		return errors.WithStack(err)
	}

	yamlString := string(yamlBytes[:])
	log.Logger.Debugf("Merged config:\n%s", yamlString)

	// todo - if the merged values are the same as the original, skip replacing the config

	// write the merged data to a temp file because we can't pipe it into kops
	tmpfile, err := ioutil.TempFile("", "kops.*.yaml")
	if err != nil {
		return errors.WithStack(err)
	}

	defer tmpfile.Close()
	defer os.Remove(tmpfile.Name()) // clean up

	if _, err := tmpfile.Write([]byte(yamlString)); err != nil {
		return errors.WithStack(err)
	}
	if err := tmpfile.Close(); err != nil {
		return errors.WithStack(err)
	}

	// Replace the cluster config
	log.Logger.Info("Replacing kops cluster config...")
	args = []string{
		"replace",
		"-f",
		tmpfile.Name(),
	}

	args = parameteriseValues(args, p.kopsConfig.Params.Global)
	args = parameteriseValues(args, p.kopsConfig.Params.Replace)

	err = utils.ExecCommand(p.kopsConfig.Binary, args, map[string]string{}, &stdoutBuf, &stderrBuf,
		"", kopsCommandTimeoutSeconds, dryRun)
	if err != nil {
		return errors.WithStack(err)
	}

	if !dryRun {
		log.Logger.Info("Kops cluster config replaced.")
	}

	log.Logger.Info("Patching instance group configs...")

	igSpecs := specs[configKeyKopsInstanceGroups].(map[interface{}]interface{})

	for instanceGroupName, newSpec := range igSpecs {
		specValues := map[string]interface{}{"spec": newSpec}
		err = p.patchInstanceGroup(p.kopsConfig, instanceGroupName.(string), specValues, dryRun)
		if err != nil {
			return errors.WithStack(err)
		}
	}

	args = []string{
		"update",
		"cluster",
		"--yes",
	}

	args = parameteriseValues(args, p.kopsConfig.Params.Global)
	args = parameteriseValues(args, p.kopsConfig.Params.UpdateCluster)

	log.Logger.Info("Updating Kops cluster...")

	err = utils.ExecCommand(p.kopsConfig.Binary, args, map[string]string{}, &stdoutBuf, &stderrBuf,
		"", 0, dryRun)
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
		&stderrBuf, "", kopsCommandTimeoutSeconds, dryRun)
	if err != nil {
		return errors.WithStack(err)
	}

	log.Logger.Debugf("Downloaded instance group config is:\n%s",
		stdoutBuf.String())

	kopsYamlConfig := map[string]interface{}{}
	err = yaml.Unmarshal(stdoutBuf.Bytes(), kopsYamlConfig)
	if err != nil {
		return errors.Wrap(err, "Error parsing kops instance group config")
	}
	log.Logger.Debugf("Yaml instance group kopsYamlConfig:\n%s", kopsYamlConfig)

	// patch in the configured spec
	err = mergo.Merge(&kopsYamlConfig, newSpec, mergo.WithOverride)
	if err != nil {
		return errors.WithStack(err)
	}

	log.Logger.Debugf("Merged instance group config is:\n%s", kopsYamlConfig)

	yamlBytes, err := yaml.Marshal(&kopsYamlConfig)
	if err != nil {
		return errors.WithStack(err)
	}

	yamlString := string(yamlBytes[:])
	log.Logger.Debugf("Merged config:\n%s", yamlString)

	// write the merged data to a temp file because we can't pipe it into kops
	tmpfile, err := ioutil.TempFile("", "kops.*.yaml")
	if err != nil {
		return errors.WithStack(err)
	}

	defer tmpfile.Close()
	defer os.Remove(tmpfile.Name()) // clean up

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
		"", kopsCommandTimeoutSeconds, dryRun)
	if err != nil {
		return errors.WithStack(err)
	}

	log.Logger.Infof("Successfully replaced config of instance group '%s'", instanceGroupName)

	return nil
}

// Converts an array of key-value parameters to CLI args
func parameteriseValues(args []string, valueMap map[string]string) []string {
	for k, v := range valueMap {
		key := strings.Replace(k, "_", "-", -1)
		args = append(args, "--"+key)

		if fmt.Sprintf("%v", v) != "" {
			args = append(args, fmt.Sprintf("%v", v))
		}
	}

	return args
}

// Parses the Kops provisioner config
func parseKopsConfig(stackConfig interfaces.IStack) (*KopsConfig, error) {
	templatedVars, err := stackConfig.TemplatedVars(nil, map[string]interface{}{})
	if err != nil {
		return nil, errors.WithStack(err)
	}
	provisionerValues, ok := templatedVars[ProvisionerKey].(map[interface{}]interface{})
	if !ok {
		return nil, errors.New("No provisioner found in stack config. You must at least set the binary path.")
	}
	log.Logger.Debugf("Marshalling: %#v", provisionerValues)

	// marshal then unmarshal the provisioner values to get the command parameters
	byteData, err := yaml.Marshal(provisionerValues)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	log.Logger.Debugf("Marshalled to: %s", string(byteData[:]))

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

	if kopsConfig.LocalPortForwardingPort == 0 {
		// set a default value if it's not set
		kopsConfig.LocalPortForwardingPort = defaultLocalPortForwardingPort
	}

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
func (p *KopsProvisioner) ensureClusterConnectivity() (bool, error) {

	if !p.needApiAccessViaBastion() || p.portForwardingActive {
		log.Logger.Infof("No need to set up SSH port forwarding to access " +
			"the API server (or it's already set up)")
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

	log.Logger.Infof("Setting up SSH port forwarding via the bastion to " +
		"the internal API server...")

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

	localPort := p.kopsConfig.LocalPortForwardingPort
	apiDomain := fmt.Sprintf("api.%s", p.kopsConfig.clusterName)

	kubeConfigPath, _ := p.stack.GetRegistry().GetString(registryKeyKubeConfig)

	if kubeConfigPath != "" {
		log.Logger.Debugf("Kubeconfig file already downloaded to '%s'", kubeConfigPath)
	} else {
		kubeConfigPath, err = p.downloadKubeConfigFile()
		if err != nil {
			return false, errors.WithStack(err)
		}

		p.stack.GetRegistry().SetString(registryKeyKubeConfig, kubeConfigPath)

		// modify the host name in the file to point to the local k8s server domain
		err = replaceAllInFile(apiDomain, fmt.Sprintf("%s:%d", kubernetesLocalHostname, localPort),
			kubeConfigPath)
		if err != nil {
			return false, errors.WithStack(err)
		}
	}

	// todo - store the kubeconfig path in the registry

	// todo - improve error handling
	go func() {
		err = p.setupPortForwarding(p.kopsConfig.SshPrivateKey, p.kopsConfig.BastionUser,
			bastionHostname, localPort, apiDomain, 443)
		if err != nil {
			log.Logger.Fatalf("An error occurred with SSH port forwarding: %v", err)
		}

		p.portForwardingActive = true
	}()

	log.Logger.Infof("Sleeping for %ds while setting up SSH port forwarding",
		sshPortForwardingDelaySeconds)
	time.Sleep(sshPortForwardingDelaySeconds * time.Second)

	return true, nil
}

// Downloads the KUBECONFIG file for the cluster to a temporary location and
// returns the path to it
func (p KopsProvisioner) downloadKubeConfigFile() (string, error) {

	log.Logger.Debugf("Downloading kubeconfig file for '%s'...",
		p.kopsConfig.clusterName)

	pattern := fmt.Sprintf("kubeconfig-%s-*", p.iStack().GetConfig().Cluster)

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
		"", kopsSleepSecondsBeforeReadyCheck, false)
	if err != nil {
		return "", errors.WithStack(err)
	}

	log.Logger.Infof("Kubeconfig file donwloaded to '%s'", kubeConfigPath)

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
func (p KopsProvisioner) setupPortForwarding(privateKey string, sshUser string, sshHost string,
	localPort int, remoteAddress string, remotePort int) error {

	connectionString := strings.Join([]string{localhost,
		strconv.Itoa(localPort), remoteAddress,
		strconv.Itoa(remotePort)}, ":")

	userHost := strings.Join([]string{sshUser, sshHost}, "@")

	log.Logger.Infof("Setting up SSH port forwarding for '%s' to '%s'",
		connectionString, userHost)

	var stdoutBuf, stderrBuf bytes.Buffer
	// todo - make this configurable. Ideally users should push a known host key
	// onto the bastion via metadata
	sshCmd := exec.Command(sshPath, "-o", "StrictHostKeyChecking no",
		"-i", privateKey, "-v", "-NL", connectionString, userHost)
	sshCmd.Stdout = &stdoutBuf
	sshCmd.Stderr = &stderrBuf

	err := sshCmd.Start()
	if err != nil {
		return errors.WithStack(err)
	}

	return nil
}

// Returns the hostname of the bastion or an empty string if it can't be found
func getBastionHostname(config *kapp.StackConfig) (string, error) {
	var stdoutBuf, stderrBuf bytes.Buffer

	query := fmt.Sprintf("LoadBalancerDescriptions["+
		"?starts_with(DNSName, `bastion-%s-`) == `true`].DNSName | [0]", config.Cluster)

	// get the bastion ELB's hostname
	err := utils.ExecCommand(awsCliPath, []string{
		"--region", config.Region,
		"elb", "describe-load-balancers",
		"--query", query,
		"--output", "text",
	}, map[string]string{}, &stdoutBuf, &stderrBuf, "",
		kopsCommandTimeoutSeconds, false)
	if err != nil {
		return "", errors.WithStack(err)
	}

	bastionHostname := strings.TrimSpace(stdoutBuf.String())

	if strings.ToLower(bastionHostname) == "none" {
		bastionHostname = ""
	}

	if bastionHostname == "" {
		log.Logger.Infof("No bastion found for cluster '%s'", config.Cluster)
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
