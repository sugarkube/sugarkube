# To do
## Repo-related tasks
* CLA (see https://github.com/heptio/ark)
* Code of conduct

## Top priorities
* Documentation!!
* Support adding some regexes to resolve whether to throw an error if certain directories/outputs exist
  depending on e.g. the provider being used. Sometimes it doesn't make sense to fail if running a kapp with the local provider because it hasn't e.g. written terraform output to a path that it would do when running with AWS, etc. Some templates (e.g. terraform backends) should only be run for remote providers, not the local one
* ~~Create a dedicated terraform installer~~  Subsumed by run units.
* ~~Create a python installer~~  Subsumed by run units.
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
  
### Cluster updates
* It should be easy to see what changes will be applied by kops - perhaps go to a two-stage approach with a '--yes' flag, to make a distinction between --dry-run and staging changes.
* If a kapp uses actions to create/update clusters, warn users if the cluster config would be altered while they're planning it, and get them to pass an extra flag to confirm the config change (to avoid accidental cluster config changes)

### Merging kapp configs
* Support passing kapp vars on the command line when only one is selected

### Kapp output
* We also need to allow access to vars from other kapps. E.g. if one kapp sets a particular variable, 
  'vars' blocks for other kapps should be able to refer to them (e.g. myvar: "{{ .kapps.somekapp.var.thevar }}")
* Provide a 'varsTemplate' field to allow for templating before parsing vars. That'll help with things like reassigning
  a map. Template this block then parse it as yaml and merge it with the other vars.
* It should be possible to load terraform outputs and use them to template other files in the kapp before installing them, without jumping through hoops with running a script to add them to the environment (a la keycloak)

### Installers
All these subsumed by run units.
* ~~Support declaring the name of makefiles in case a repo already has one~~
* ~~Get rid of the duplication of mapping variables - we currently do it once in sugarkube.yaml files then again in makefiles. Try to automate the mapping in makefiles~~
* ~~Need to use 'override' with params in makefiles. How can we make that simpler?~~
* ~~See if we can suppress warning in overridden makefiles by using the technique by mpb~~ [described here](https://stackoverflow.com/questions/11958626/make-file-warning-overriding-commands-for-target)
* ~~document  tf-params vs tf-opts and the same for helm in the makefiles~~
* ~~Create a generic 'script' installer that takes the name of the script to run and whether to pass it values as env vars or written to a file as parameters~~
* ~~This could support ruby, js, python, etc. May need an 'init' call to e.g. make things install dependencies~~

### Developer experience
* Stream console output in real-time - see stern for an example of streaming logs from multiple processes in parallel. Add a flag to enable this.
* use ps (https://github.com/shirou/gopsutil/) to check whether SSH port forwarding is actually set up, and 
  if not set it up again. Also, when sugarkube is invoked throw an error if port forwarding is already set up
* Or do ssh using a golang library so we can make it more robust (reconnecting on dropped connections, etc)
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

* Don't throw an error running `sugarkube version` if no config file could be loaded

## Other things to consider
* Is being focussed on clusters a mistake? 
    * We could help provision other hosted services, e.g. ElastiCache, BigQuery, etc. 
    * Maybe we should think more in terms of a 'context', e.g. dev1 could be used to run different related
      resources but with a different identifier to e.g. segregate a VPC from an ElastiCache or something (in 2 
      different stacks)
  