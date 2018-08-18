package provisioner

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/sugarkube/sugarkube/internal/pkg/log"
	"github.com/sugarkube/sugarkube/internal/pkg/provider"
	"github.com/sugarkube/sugarkube/internal/pkg/vars"
	"os/exec"
	"strings"
)

type MinikubeProvisioner struct {
	Provisioner
}

// todo - make configurable
const MINIKUBE_PATH = "minikube"

func (p MinikubeProvisioner) Create(sc *vars.StackConfig, values provider.Values, dryRun bool) error {

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
		log.Infof("Launching Minikube cluster... Executing %s %s", MINIKUBE_PATH,
			strings.Join(args, " "))

		err := cmd.Run()

		if err != nil {
			return errors.Wrap(err, "Failed to start a Minikube cluster")
		}

		log.Infof("Minikube cluster successfully started")
	}

	return nil
}

func (p MinikubeProvisioner) IsOnline(sc *vars.StackConfig, values provider.Values) (bool, error) {
	panic("not implemented")
}

func (p MinikubeProvisioner) Update(sc *vars.StackConfig, values provider.Values) error {
	panic("not implemented")
}
