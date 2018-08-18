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

func (p MinikubeProvisioner) Create(sc *vars.StackConfig, values provider.Values) error {

	log.Debugf("Creating stack with Minikube and values: %#v", values)

	args := make([]string, 0)
	args = append(args, "start")

	provisionerValues := values[PROVISIONER_KEY].(map[interface{}]interface{})

	for k, v := range provisionerValues {
		key := strings.Replace(k.(string), "_", "-", -1)
		args = append(args, "--"+key)
		args = append(args, fmt.Sprintf("%v", v))
	}

	log.Infof("Launching Minikube cluster with args %s", strings.Join(args, " "))

	cmd := exec.Command("minikube", args...)

	// todo - pass in
	dryRun := true

	if dryRun {
		log.Infof("Dry run. Skipping invoking Minikube.")
	} else {
		err := cmd.Run()

		if err != nil {
			return errors.Wrap(err, "Failed to start a Minikube cluster")
		}
	}

	return nil
}

func (p MinikubeProvisioner) IsOnline(sc *vars.StackConfig, values provider.Values) (bool, error) {
	panic("not implemented")
}

func (p MinikubeProvisioner) Update(sc *vars.StackConfig, values provider.Values) error {
	panic("not implemented")
}
