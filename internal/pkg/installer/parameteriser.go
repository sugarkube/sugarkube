package installer

import "github.com/sugarkube/sugarkube/internal/pkg/kapp"

// This is a generic way of inspecting kapps to see what they contain and what
// env vars/CLI parameters should be passed to their installers

// todo - move into a default config file, but allow them to be overridden/
// addtional interfaces defined.
var parameteriserConfig = `
kapp_interfaces:	# different things a kapp might contain. A kapp may 'implement' 
					# multiple interfaces (e.g. contain both a helm chart and terraform configs
  helm_chart:
    heuristics: 			# inspections we can carry out on a kapp to see what it contains
    - file:
        pattern: Chart.yaml		# regex to search for under the kapp root dir
    params:
      env_vars:
      - name: KUBE_CONTEXT
        value: 
          type: vars_lookup
          path: provider
          key: kube_context
      - name: NAMESPACE
        value: 
          type: obj_field
          path: kapp
          key: Id
      - name: RELEASE
        value: 
          type: obj_field
          path: kapp
          key: Id
      cli_arg:
        name: helm-opts
        components:
        - key: -f
          value
            pattern: values-(\w+).yaml

  k8s_resource:             # a naked k8s resource. No heuristics. Expect to find
    params:					# it listed in 'sugarkube.yaml'
      env_vars:
      - name: KUBE_CONTEXT
        value: 
          type: vars_lookup
          path: provider
          key: kube_context

  terraform:
    heuristics:
    - file:
        pattern: terraform.*
        type: dir
    params:
      cli_arg:
        name: tf-opts
        components:
        - key: -var-file
          value:
            pattern: vars/(\w+).tfvars
`

// Examines a kapp to find out what it contains, and therefore what env vars/
// CLI args need passing to it by an Installer.
func identifyKappInterfaces(kappObj kapp.Kapp) []string {
	return nil
}
