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
	"github.com/sugarkube/sugarkube/internal/pkg/kapp"
	"github.com/sugarkube/sugarkube/internal/pkg/log"
	"github.com/sugarkube/sugarkube/internal/pkg/provider"
	"github.com/sugarkube/sugarkube/internal/pkg/utils"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
)

// todo - make configurable
const KOPS_PATH = "kops"

// number of seconds to timeout after while running kops commands (apart from updates)
const KOPS_COMMAND_TIMEOUT_SECONDS = 30

// number of seconds to sleep after the cluster has come online before checking whether
// it's ready
const KOPS_SLEEP_SECONDS_BEFORE_READY_CHECK = 60

const KOPS_SPECS_KEY = "specs"
const KOPS_CLUSTER_KEY = "cluster"
const KOPS_INSTANCE_GROUPS_KEY = "instanceGroups"

type KopsProvisioner struct {
	clusterSot clustersot.ClusterSot
}

type KopsConfig struct {
	Params struct {
		Global            map[string]string
		CreateCluster     map[string]string `yaml:"create_cluster"`
		UpdateCluster     map[string]string `yaml:"update_cluster"`
		GetClusters       map[string]string `yaml:"get_clusters"`
		GetInstanceGroups map[string]string `yaml:"get_instance_groups"`
		RollingUpdate     map[string]string `yaml:"rolling_update"`
		Replace           map[string]string
	}
}

func (p KopsProvisioner) ClusterSot() (clustersot.ClusterSot, error) {
	if p.clusterSot == nil {
		clusterSot, err := clustersot.NewClusterSot(clustersot.KUBECTL)
		if err != nil {
			return nil, errors.WithStack(err)
		}

		p.clusterSot = clusterSot
	}

	return p.clusterSot, nil
}

// Returns a bool indicating whether the cluster configuration has already been created
func (p KopsProvisioner) clusterConfigExists(sc *kapp.StackConfig, providerImpl provider.Provider) (bool, error) {

	providerVars := provider.GetVars(providerImpl)
	log.Logger.Debugf("Checking if a Kops cluster config exists for values: %#v", providerVars)

	provisionerValues := providerVars[PROVISIONER_KEY].(map[interface{}]interface{})
	kopsConfig, err := getKopsProvisionerConfig(provisionerValues)
	if err != nil {
		return false, errors.WithStack(err)
	}

	args := []string{"get", "clusters"}
	args = parameteriseValues(args, kopsConfig.Params.Global)
	args = parameteriseValues(args, kopsConfig.Params.GetClusters)

	var stdoutBuf, stderrBuf bytes.Buffer

	err = utils.ExecCommand(KOPS, args, &stdoutBuf, &stderrBuf, "",
		KOPS_COMMAND_TIMEOUT_SECONDS, false)
	if err != nil {
		if errors.Cause(err) == context.DeadlineExceeded {
			return false, errors.Wrap(err,
				"Timed out trying to retrieve kops cluster config. "+
					"Check your credentials.")
		}

		if _, ok := errors.Cause(err).(*exec.ExitError); ok {
			log.Logger.Debug("Cluster config doesn't exist")
			return false, nil
		} else {
			return false, errors.Wrap(err, "Error fetching kops clusters")
		}
	}

	return true, nil
}

// Creates a Kops cluster config. Note: This doesn't actually launch a Kops cluster,
// that only happens when 'kops update' is run.
func (p KopsProvisioner) create(sc *kapp.StackConfig, providerImpl provider.Provider,
	dryRun bool) error {

	providerVars := provider.GetVars(providerImpl)
	log.Logger.Debugf("Provider vars: %#v", providerVars)

	provisionerValues := providerVars[PROVISIONER_KEY].(map[interface{}]interface{})
	kopsConfig, err := getKopsProvisionerConfig(provisionerValues)
	if err != nil {
		return errors.WithStack(err)
	}

	args := []string{"create", "cluster"}
	args = parameteriseValues(args, kopsConfig.Params.Global)
	args = parameteriseValues(args, kopsConfig.Params.CreateCluster)

	var stdoutBuf, stderrBuf bytes.Buffer

	log.Logger.Info("Creating Kops cluster config...")
	err = utils.ExecCommand(KOPS_PATH, args, &stdoutBuf, &stderrBuf,
		"", KOPS_COMMAND_TIMEOUT_SECONDS, dryRun)
	if err != nil {
		return errors.WithStack(err)
	}

	if !dryRun {
		log.Logger.Debugf("Kops returned:\n%s", stdoutBuf.String())
		log.Logger.Infof("Kops cluster config created")
	}

	err = p.patch(sc, providerImpl, dryRun)
	if err != nil {
		return errors.WithStack(err)
	}

	sc.Status.StartedThisRun = true
	// only sleep before checking the cluster fo readiness if we started it
	sc.Status.SleepBeforeReadyCheck = KOPS_SLEEP_SECONDS_BEFORE_READY_CHECK

	return nil
}

// Returns a boolean indicating whether the cluster is already online
func (p KopsProvisioner) isAlreadyOnline(sc *kapp.StackConfig, providerImpl provider.Provider) (bool, error) {
	configExists, err := p.clusterConfigExists(sc, providerImpl)
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

	online, err := clustersot.IsOnline(clusterSot, sc, providerImpl)
	if err != nil {
		return false, errors.WithStack(err)
	}

	return online, nil
}

// No-op function, required to fully implement the Provisioner interface
func (p KopsProvisioner) update(sc *kapp.StackConfig, providerImpl provider.Provider,
	dryRun bool) error {

	err := p.patch(sc, providerImpl, dryRun)
	if err != nil {
		return errors.WithStack(err)
	}

	providerVars := provider.GetVars(providerImpl)

	provisionerValues := providerVars[PROVISIONER_KEY].(map[interface{}]interface{})
	kopsConfig, err := getKopsProvisionerConfig(provisionerValues)
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

	args = parameteriseValues(args, kopsConfig.Params.Global)
	args = parameteriseValues(args, kopsConfig.Params.RollingUpdate)

	var stdoutBuf, stderrBuf bytes.Buffer

	log.Logger.Info("Running Kops rolling update...")
	err = utils.ExecCommand(KOPS_PATH, args, &stdoutBuf, &stderrBuf,
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
func (p KopsProvisioner) patch(sc *kapp.StackConfig, providerImpl provider.Provider,
	dryRun bool) error {
	configExists, err := p.clusterConfigExists(sc, providerImpl)
	if err != nil {
		return errors.WithStack(err)
	}

	// can't update a non-existent config
	if !configExists {
		return nil
	}

	providerVars := provider.GetVars(providerImpl)
	provisionerValues := providerVars[PROVISIONER_KEY].(map[interface{}]interface{})
	kopsConfig, err := getKopsProvisionerConfig(provisionerValues)
	if err != nil {
		return errors.WithStack(err)
	}

	// get the kops config
	args := []string{
		"get",
		"clusters",
		"-o",
		"yaml",
	}

	args = parameteriseValues(args, kopsConfig.Params.Global)
	args = parameteriseValues(args, kopsConfig.Params.GetClusters)

	var stdoutBuf, stderrBuf bytes.Buffer

	log.Logger.Debug("Downloading config for kops cluster...")

	err = utils.ExecCommand(KOPS_PATH, args, &stdoutBuf, &stderrBuf,
		"", KOPS_COMMAND_TIMEOUT_SECONDS, dryRun)
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

	specs, err := convert.MapInterfaceInterfaceToMapStringInterface(
		provisionerValues[KOPS_SPECS_KEY].(map[interface{}]interface{}))
	if err != nil {
		return errors.WithStack(err)
	}

	clusterSpecs := specs[KOPS_CLUSTER_KEY]

	specValues := map[string]interface{}{"spec": clusterSpecs}

	log.Logger.Debugf("Spec to merge in:\n%s", specValues)

	// patch in the configured spec
	mergo.Merge(&kopsYamlConfig, specValues, mergo.WithOverride)

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
		// todo - either remove this use of log.Logger.Fatal and return an error, or use it throughout
		log.Logger.Fatal(err)
	}

	defer os.Remove(tmpfile.Name()) // clean up

	if _, err := tmpfile.Write([]byte(yamlString)); err != nil {
		tmpfile.Close()
		log.Logger.Fatal(err)
	}
	if err := tmpfile.Close(); err != nil {
		log.Logger.Fatal(err)
	}

	// Replace the cluster config
	log.Logger.Info("Replacing kops cluster config...")
	args = []string{
		"replace",
		"-f",
		tmpfile.Name(),
	}

	args = parameteriseValues(args, kopsConfig.Params.Global)
	args = parameteriseValues(args, kopsConfig.Params.Replace)

	err = utils.ExecCommand(KOPS_PATH, args, &stdoutBuf, &stderrBuf,
		"", KOPS_COMMAND_TIMEOUT_SECONDS, dryRun)
	if err != nil {
		return errors.WithStack(err)
	}

	if !dryRun {
		log.Logger.Info("Kops cluster config replaced.")
	}

	log.Logger.Debug("Patching instance group configs...")

	igSpecs := specs[KOPS_INSTANCE_GROUPS_KEY].(map[interface{}]interface{})

	for instanceGroupName, newSpec := range igSpecs {
		specValues := map[string]interface{}{"spec": newSpec}
		p.patchInstanceGroup(kopsConfig, instanceGroupName.(string), specValues)
	}

	args = []string{
		"update",
		"cluster",
		"--yes",
	}

	args = parameteriseValues(args, kopsConfig.Params.Global)
	args = parameteriseValues(args, kopsConfig.Params.UpdateCluster)

	log.Logger.Info("Updating Kops cluster...")

	err = utils.ExecCommand(KOPS_PATH, args, &stdoutBuf, &stderrBuf,
		"", 0, dryRun)
	if err != nil {
		return errors.WithStack(err)
	}

	return nil
}

func (p KopsProvisioner) patchInstanceGroup(kopsConfig *KopsConfig, instanceGroupName string,
	newSpec map[string]interface{}) error {
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
	err := utils.ExecCommand(KOPS_PATH, args, &stdoutBuf, &stderrBuf,
		"", KOPS_COMMAND_TIMEOUT_SECONDS, false)
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
	mergo.Merge(&kopsYamlConfig, newSpec, mergo.WithOverride)

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
		log.Logger.Fatal(err)
	}

	defer os.Remove(tmpfile.Name()) // clean up

	if _, err := tmpfile.Write([]byte(yamlString)); err != nil {
		tmpfile.Close()
		log.Logger.Fatal(err)
	}
	if err := tmpfile.Close(); err != nil {
		log.Logger.Fatal(err)
	}

	// replace the cluster config

	log.Logger.Debugf("Replacing config of Kops instance group %s...", instanceGroupName)
	args = []string{
		"replace",
		"-f",
		tmpfile.Name(),
	}

	args = parameteriseValues(args, kopsConfig.Params.Global)
	args = parameteriseValues(args, kopsConfig.Params.Replace)

	err = utils.ExecCommand(KOPS_PATH, args, &stdoutBuf, &stderrBuf,
		"", KOPS_COMMAND_TIMEOUT_SECONDS, false)
	if err != nil {
		return errors.WithStack(err)
	}

	log.Logger.Infof("Successfully replaced config of instance group %s", instanceGroupName)

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
func getKopsProvisionerConfig(provisionerValues map[interface{}]interface{}) (*KopsConfig, error) {
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

	return &kopsConfig, nil
}
