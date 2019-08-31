# To do
## Repo-related tasks
* CLA (see https://github.com/heptio/ark)
* Code of conduct

## Top priorities
* The `kapps clean` command doesn't work - probably not merging in run units from the global config properly
* The kapps validate command should search for the actual configured command, not assume it's the same name as the requirement itself. Test it with the wordpress kapp.
* The kapps validate command should make sure that all run steps are uniquely named to avoid issues calling different ones
* Maybe the validate command should be implicitly run before `kapps install/delete`. The problem is with executing actions - if we default to running them it's dangerous, and if we default to not running them then perhaps config changes might also accidentally be applied perhaps or the commands run as expected. I think we should make `validate` search for run actions. If any are found we should require users to explictly either pass `--run-actions/--run-*-actions` or `--skip-*-actions` so they're actively choosing what to do.
* The `kapps validate` command should be strict and throw an error if any unexpected yaml keys are present 
* The Wordpress kapp should provide control over whether to install fixtures or not. Maybe by default it should only do it when the kapp is first installed, since after that it fails...
* Update the prometheus-operator kapp to delete its CRDs when it's deleted
* Add an '--only' option to the 'kapps' subcommands to only process marked nodes. Outputs will not be loaded for unmarked nodes/dependencies. This will speed up kapp development when you're iterating on a specific kapp and don't want to wait for terraform to load outputs for a kapp you don't care about. 
* Only run a kops update if the spec has changed (diff the new spec with the existing one)
* Throw a more useful error if AWS creds have expired (e.g. for kops or trying to set up cluster connectivity) (created https://github.com/kubernetes/kops/issues/7393)
* Documentation
  * Document the dangers of adding provider vars dirs (i.e. that the next time sugarkube is run it'll replace the config). It should only be used in certain situations (and probably never in prod)
* Add a way of replacing kapp settings in stack configs (e.g. to replace dependencies)
* Support defaults at the stack level (e.g. to pin helm/kubectl binaries per stack)
* Add a setting to throw an error if kapp IDs aren't globally unique. We don't care, but terraform does with our sample naming convention. The options are either to add the manifest ID to the TF state path which stops people reorganising, or making kapp IDs globally unique, otherwise e.g. 2 wordpress instances in different manifests could clobber each other  
* Setting 'versions' in stacks fails when there are 2 references to the same kapp (but different sources)
* Source URIs without branches should be ignored (unless an extra flag is set) to make it easy to ignore them in a stack by not setting a branch (it's safest to ignore them)
* Run units defined in kapps should be merged with those in the main config file, so only specific units can be overridden and the configured defaults used for other units. At the moment all units must be redefined even if on a single unit is needed (see cert manager)
* Add flags to selectively skip/include running specific run steps (some steps - e.g. helm install - can be slow, which is annoying if you're debugging a later run step)
* It should be possible to set kapp vars that are maps
* Think of a good way of declaring per-project names that can be used for namespacing (i.e. to allow multiple clusters to be brought up for different reasons)
* Add a flag on the 'kapps install' command to print out what it would execute for each step to make it easier to debug what would happen (so users don't need to enable logging to see the commands that would be executed)
* `ws create` should support `-i/-x` selectors to support selectively updating kapps

### Cluster updates
* It should be easy to see what changes will be applied by kops - perhaps go to a two-stage approach with a '--yes' flag, to make a distinction between --dry-run and staging changes.

### Merging kapp configs
* Support passing kapp vars on the command line when only one is selected

### Kapp output
* We also need to allow access to vars from other kapps. E.g. if one kapp sets a particular variable, 
  'vars' blocks for other kapps should be able to refer to them (e.g. myvar: "{{ .kapps.somekapp.var.thevar }}")
* Provide a 'varsTemplate' field to allow for templating before parsing vars. That'll help with things like reassigning
  a map. Template this block then parse it as yaml and merge it with the other vars.
* It should be possible to load terraform outputs and use them to template other files in the kapp before installing them, without jumping through hoops with running a script to add them to the environment (a la keycloak)

### Developer experience
* Stream console output in real-time - see stern for an example of streaming logs from multiple processes in parallel. Add a flag to enable this.
* Make SSH connections more robust (reconnect on dropped connections, etc)
* Improve the UX around using caches/workspaces, especially re updating while working on a change (sugarkube shouldn't bomb out but should update whatever it can)
  
### Everything else
* Support declaring templates as 'sensitive' - they should be templated just-in-time then deleted (even on error/interrupts)

* Support acquiring manifests with the acquirers (to support pulling from git repos) - this will help multi-team setups, where the platform team can 
  maintain the main stack config, pulling in manifests from repos the app teams have access to (so they don't need
  access to the main config repo). Manifest variables will simplify passing env vars to all kapps in the manifest
  (e.g. for the tiller-namespace, etc.)

* Add support for verifying signed tags
* More tests 
* Fix failing integration test

* ~~Consider adding a cache so we can do cluster diffing to only install kapps that have changed to speed up
  deploying changes. Use a ClusterSOT for that.~~
* Create a cache manager whose job it is to organise where files are stored in a cache to enable the no-op provisioner to be used

## Other things to consider
* Is being focussed on clusters a mistake? 
    * We could help provision other hosted services, e.g. ElastiCache, BigQuery, etc. 
    * Maybe we should think more in terms of a 'context', e.g. dev1 could be used to run different related
      resources but with a different identifier to e.g. segregate a VPC from an ElastiCache or something (in 2 
      different stacks)
  