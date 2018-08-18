package clustersot

import (
	"github.com/sugarkube/sugarkube/internal/pkg/provider"
	"github.com/sugarkube/sugarkube/internal/pkg/vars"
	"os/exec"
)

type KubeCtlClusterSot struct {
	ClusterSot
}

// todo - make configurable
const KUBECTL_PATH = "kubectl"

func (c KubeCtlClusterSot) IsReady(sc *vars.StackConfig, values provider.Values) (bool, error) {

	context := values["kube_context"].(string)

	// poll `kubectl --context {{ kube_context }} get namespace`
	cmd := exec.Command(KUBECTL_PATH, "--context", context, "get", "namespace")
	err := cmd.Run()
}
