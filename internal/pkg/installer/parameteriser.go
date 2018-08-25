package installer

import (
	"github.com/pkg/errors"
	"github.com/sugarkube/sugarkube/internal/pkg/kapp"
)

// This is a generic way of inspecting kapps to see what they contain and what
// env vars/CLI parameters should be passed to their installers

// todo - move into a default config file, but allow them to be overridden/
// addtional interfaces defined.
//var parameteriserConfig = `
//kapp_interfaces:	# different things a kapp might contain. A kapp may 'implement'
//					# multiple interfaces (e.g. contain both a helm chart and terraform configs
//  helm_chart:
//    heuristics: 			# inspections we can carry out on a kapp to see what it contains
//    - file:
//        pattern: Chart.yaml		# regex to search for under the kapp root dir
//    params:
//      env:
//      - name: KUBE_CONTEXT
//        value:
//          type: vars_lookup
//          path: provider
//          key: kube_context
//      - name: NAMESPACE		# default value. Allow overriding it in the installer config.
//        value: 				# think of how to configure that here...
//          type: obj_field
//          path: kapp
//          key: Id
//      - name: RELEASE
//        value:
//          type: obj_field
//          path: kapp
//          key: Id
//      cliArgs:
//      - name: helm-opts
//        components:
//        - key: -f
//          value
//            pattern: values-(\w+).yaml
//
//  k8s_resource:             # a naked k8s resource. No heuristics. Expect to find
//    params:					# it listed in 'sugarkube.yaml'
//      env:
//      - name: KUBE_CONTEXT
//        value:
//          type: vars_lookup
//          path: provider
//          key: kube_context
//
//  terraform:
//    heuristics:
//    - file:
//        pattern: terraform.*
//        type: dir
//    params:
//      cliArgs:
//      - name: tf-opts
//        components:			# by default collapse multiple values into a
//        - key: -var-file		# single CLI arg
//          value:
//            pattern: vars/(\w+).tfvars
//`

const IMPLEMENTS_HELM = "helm"
const IMPLEMENTS_TERRAFORM = "terraform"
const IMPLEMENTS_K8S = "k8s"

// Examines a kapp to find out what it contains, and therefore what env vars/
// CLI args need passing to it by an Installer.
func identifyKappInterfaces(kappObj *kapp.Kapp) ([]string, error) {
	// todo - parse the above config and test the kapp using it.
	// todo - also look in the kapp's sugarkube.yaml file if it exists

	interfaces := make([]string, 0)

	// todo - remove IMPLEMENTS_K8S from this. It's a temporary kludge until we
	// can get it from the kapp's sugarkube.yaml file
	interfaces = append(interfaces, IMPLEMENTS_K8S)

	// todo - remove this kludge to find out whether the kapp contains a helm chart.
	chartPaths, err := findFilesByPattern(kappObj.RootDir, "Chart.yaml", true)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	if len(chartPaths) > 0 {
		interfaces = append(interfaces, IMPLEMENTS_HELM)
	}

	// todo - remove this kludge to find out whether the kapp contains terraform configs
	terraformPaths, err := findFilesByPattern(kappObj.RootDir, "terraform", true)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	if len(terraformPaths) > 0 {
		interfaces = append(interfaces, IMPLEMENTS_TERRAFORM)
	}

	return interfaces, nil
}
