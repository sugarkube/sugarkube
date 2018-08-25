package installer

import "github.com/sugarkube/sugarkube/internal/pkg/kapp"

// This is a generic way of inspecting kapps to see what they contain and what
// env vars/CLI parameters should be passed to their installers

// todo - move into config
var parameteriserConfig = `
kapp_interfaces:			# different things a kapp might contain. May be multiple.
  helm_chart:
    heuristics: 			# inspections we can carry out on a kapp to see what it contains
    - file:
        glob: Chart.yaml
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
        glob: terraform*
        type: dir
    params:
      cli_arg:
        name: tf-opts
        components:
        - key: -var-file
          value:
            pattern: vars/(\w+).tfvars
`

func identifyKappInterfaces(kappObj kapp.Kapp) []string {
	return nil
}
