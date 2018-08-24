package provisioner

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/sugarkube/sugarkube/internal/pkg/kapp"
	"github.com/sugarkube/sugarkube/internal/pkg/log"
	"github.com/sugarkube/sugarkube/internal/pkg/provider"
	"github.com/sugarkube/sugarkube/internal/pkg/provisioner/clustersot"
	"time"
)

type Provisioner interface {
	// Returns the ClusterSot for this provisioner
	ClusterSot() (clustersot.ClusterSot, error)
	// Creates a cluster
	create(sc *kapp.StackConfig, values provider.Values, dryRun bool) error
	// Returns whether the cluster is already running
	isAlreadyOnline(sc *kapp.StackConfig, values provider.Values) (bool, error)
	// Update the cluster config if supported by the provisioner
	update(sc *kapp.StackConfig, values provider.Values) error
}

// key in Values that relates to this provisioner
const PROVISIONER_KEY = "provisioner"

// Implemented provisioner names
const MINIKUBE = "minikube"
const KOPS = "kops"

// Factory that creates providers
func NewProvisioner(name string) (Provisioner, error) {
	if name == MINIKUBE {
		return MinikubeProvisioner{}, nil
	}

	if name == KOPS {
		return KopsProvisioner{}, nil
	}

	return nil, errors.New(fmt.Sprintf("Provisioner '%s' doesn't exist", name))
}

// Creates a cluster using an implementation of a Provisioner
func Create(p Provisioner, sc *kapp.StackConfig, values provider.Values, dryRun bool) error {
	return p.create(sc, values, dryRun)
}

// Return whether the cluster is already online
func IsAlreadyOnline(p Provisioner, sc *kapp.StackConfig, values provider.Values) (bool, error) {

	log.Infof("Checking whether cluster '%s' is already online...", sc.Cluster)

	online, err := p.isAlreadyOnline(sc, values)
	if err != nil {
		return false, errors.WithStack(err)
	}

	if online {
		log.Infof("Cluster '%s' is online", sc.Cluster)
	} else {
		log.Infof("Cluster '%s' is not online", sc.Cluster)
	}

	sc.Status.IsOnline = online
	return online, nil
}

// Wait for a cluster to come online, then to become ready.
func WaitForClusterReadiness(p Provisioner, sc *kapp.StackConfig, values provider.Values) error {
	clusterSot, err := p.ClusterSot()
	if err != nil {
		return errors.WithStack(err)
	}

	log.Infof("Checking whether the cluster is online... Will try for %d seconds",
		sc.OnlineTimeout)

	clusterWasOffline := false

	timeoutTime := time.Now().Add(time.Second * time.Duration(sc.OnlineTimeout))
	for time.Now().Before(timeoutTime) {
		online, err := clustersot.IsOnline(clusterSot, sc, values)
		if err != nil {
			return errors.WithStack(err)
		}

		if online {
			log.Info("Cluster is online")
			break
		} else {
			clusterWasOffline = true
			log.Info("Cluster isn't online. Sleeping...")
			time.Sleep(time.Duration(5) * time.Second)
		}
	}

	if !sc.Status.IsOnline {
		return errors.New("Timed out waiting for the cluster to come online")
	}

	// only sleep before checking readiness if the cluster was initially offline
	sleepTime := sc.Status.SleepBeforeReadyCheck
	if clusterWasOffline || sc.Status.StartedThisRun && sleepTime > 0 {
		log.Infof("Sleeping for %d seconds before checking cluster readiness...", sleepTime)
		time.Sleep(time.Second * time.Duration(sleepTime))
	}

	log.Infof("Checking whether the cluster is ready...")

	readinessTimeoutTime := time.Now().Add(time.Second * time.Duration(sc.OnlineTimeout))
	for time.Now().Before(readinessTimeoutTime) {
		ready, err := clustersot.IsReady(clusterSot, sc, values)
		if err != nil {
			return errors.WithStack(err)
		}

		if ready {
			log.Info("Cluster is ready")
			break
		} else {
			log.Info("Cluster isn't ready. Sleeping...")
			time.Sleep(time.Duration(5) * time.Second)
		}
	}

	if !sc.Status.IsReady {
		return errors.New("Timed out waiting for the cluster to become ready")
	}

	return nil
}
