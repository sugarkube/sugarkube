# Changelog
## 0.7.0
* Renamed the `kapps apply` subcommand to `kapps install` and `kapps destroy` to `kapps delete`
* Renamed the `destroy` make target and updated the [common makefiles](https://github.com/sugarkube/kapps/tree/master/incubator/common-makefiles)
* Changed the '--approved' flag to '--yes' to make it more intuitive
* Add a command `sugarkube completions` to generate a bash completions script
* SSH port forwarding tunnels are now closed (or an attempt is made to close them) after commands finish, including if errors occur or the program is killed via a signal.
* Paths to templates declared in kapps are relative to the directory containing the `sugarkube.yaml` file
* The strategy for merging list values under the same map key is now configurable. By default list values under the same map key will be appended. Set the `overwrite-merged-lists` setting to `true` to have higher priority lists completely replace the contents of lower priority lists.
* Provider vars are re-evaluated for each tranche of a plan allowing for registry values to modify the config
* Provider vars files now allow limited templating to allow registry values to conditionally affect the provider config (e.g. `{{ if .registry.outputs.<blah> }}...{{ end }}` )
* Kapp outputs are now parsed and added to the registry after they've finished running
* Sensitive kapp outputs will be deleted as soon as the output has been parsed and added to the registry
* By default missing template destination directories won't cause Sugarkube to abort. But it can be made to by setting `dest_dir_must_exist: true` in a template definition.
* Add an option to CLI commands to fail if template destination directories don't exist
* post_actions is now a list of maps. See documentation.
* Kapps can now push additional provider vars dirs onto the list that will be merged by a provider. This allows them to modify the provider's config.
* Kapp templates now get rendered before and after installing/deleting kapps so they can use their own output in templates
* Default variables can now be defined per program in the global sugarkube-conf.yaml file. Keys map to programs in a kapp's 'requires' block

## 0.6.0 (25/3/19)
* Major code clean up & refactoring
* Variables can now be interpolated based on other variables
* Kapp variables are now namespaced under a dedicated key ('kapp') to prevent them overwriting system variables
* Added a command `kapps vars` for inspecting variables available to a kapp
* Added a command `cluster vars` for inspecting variables available for a cluster/stack
* Unified the URI format for kapp sources
* Allow settings for kapps to be overridden from stack config files
* Custom provisioner binaries can now be specified per stack to control which version of a provisioner is used in each stack
* Improved error handling
* Make some CLI options required positional arguments
* Clarify the extent to which values can be supplied on the command line vs in stack config files
* Rename the `name` key on sources to `id` for consistency with everywhere else
* Allow only selected kapps to be installed
* Templates for kapps are written immediately before applying kapps by default now
* Kapps can now declare actions to be run after applying the kapp. Currently the only supported post action is to update a cluster (the cluster will be launched if it's offline)
* Removed support for init manifests. All manifests should be run and they're expected to be idempotent.
* Depend on forked version of go-yaml that doesn't split output wider than 80 characters
* Kapp template source/dest paths can now contain variables
* Kapp vars are namespaced under the `.kapp.vars` key in the stack config map
* Provider vars are not namespaced in the stack config map
* Provider vars directories will be searched breadth-first in a similar way to how kapp vars dirs are searched to allow cross-cutting configuration (e.g. all AMI IDs for a region can be set once and will be merged into each region's config)
* Add a way of connecting to K8s API servers that were created with internal load balancers by kops (provided there's a bastion) 
* Allow the path to a custom config file to be given
* Enable and use trace-level logging to make debug level logging easier to read
* Change the format of the 'args' YAML in kapps

## 0.5.0 (2/12/19)
* Kapps need to declare what environment variables they want and what to map to them in a 'sugarkube.yaml' file

## 0.4.0 (29/9/18)
* Neaten up logging

## 0.3.0 (28/9/18)
* Allow templates to be rendered for kapps
* Variables can be loaded from various directories

## 0.2.0 (22/9/18)
* Added an AWS provider
* Added a KOPS provisioner

## 0.1.1 (28/8/18)
* Pass the user's environment variables when running commands

## 0.1.0 (25/8/18)
Initial release