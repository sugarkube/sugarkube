package provisioner

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/sugarkube/sugarkube/internal/pkg/clustersot"
	"github.com/sugarkube/sugarkube/internal/pkg/kapp"
	"github.com/sugarkube/sugarkube/internal/pkg/log"
	"github.com/sugarkube/sugarkube/internal/pkg/provider"
	"os/exec"
	"strings"
)

type MinikubeProvisioner struct {
	clusterSot clustersot.ClusterSot
}

// todo - make configurable
const MINIKUBE_PATH = "minikube"

// todo read docs re `minikube profile` to run multiple instances on the same host

// Seconds to sleep after the cluster is online but before checking whether it's ready.
// This gives pods a chance to be launched. If we check immediately there are no pods.
const SLEEP_SECONDS_BEFORE_READY_CHECK = 30

func (p MinikubeProvisioner) ClusterSot() (clustersot.ClusterSot, error) {
	if p.clusterSot == nil {
		clusterSot, err := clustersot.NewClusterSot(clustersot.KUBECTL)
		if err != nil {
			return nil, errors.WithStack(err)
		}

		p.clusterSot = clusterSot
	}

	return p.clusterSot, nil
}

// Creates a new minikube cluster
func (p MinikubeProvisioner) create(sc *kapp.StackConfig, values provider.Values, dryRun bool) error {

	log.Debugf("Creating stack with Minikube and values: %#v", values)

	args := make([]string, 0)
	args = append(args, "start")

	provisionerValues := values[PROVISIONER_KEY].(map[interface{}]interface{})

	for k, v := range provisionerValues {
		key := strings.Replace(k.(string), "_", "-", -1)
		args = append(args, "--"+key)
		args = append(args, fmt.Sprintf("%v", v))
	}

	cmd := exec.Command(MINIKUBE_PATH, args...)

	if dryRun {
		log.Infof("Dry run. Skipping invoking Minikube, but would execute: %s %s",
			MINIKUBE_PATH, strings.Join(args, " "))
	} else {
		log.Infof("Launching Minikube cluster... Executing: %s %s", MINIKUBE_PATH,
			strings.Join(args, " "))

		err := cmd.Run()

		if err != nil {
			return errors.Wrap(err, "Failed to start a Minikube cluster")
		}

		log.Infof("Minikube cluster successfully started")
	}

	sc.Status.StartedThisRun = true
	// only sleep before checking the cluster fo readiness if we started it
	sc.Status.SleepBeforeReadyCheck = SLEEP_SECONDS_BEFORE_READY_CHECK

	return nil
}

// Returns whether a minikube cluster is already online
func (p MinikubeProvisioner) isAlreadyOnline(sc *kapp.StackConfig, values provider.Values) (bool, error) {
	cmd := exec.Command(MINIKUBE_PATH, "status")
	err := cmd.Run()

	if err != nil {
		// assume no cluster is up if the command starts but doesn't complete successfully
		if _, ok := err.(*exec.ExitError); ok {
			return false, nil
		} else {
			// something else, so return an error
			return false, errors.WithStack(err)
		}
	}

	// otherwise assume a cluster is online
	return true, nil
}

// No-op function, required to fully implement the Provisioner interface
func (p MinikubeProvisioner) update(sc *kapp.StackConfig, values provider.Values) error {
	log.Infof("Updating minikube clusters has no effect. Ignoring.")
	return nil
}
