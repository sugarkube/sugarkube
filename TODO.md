# To do
## Repo-related tasks
* CLA (see https://github.com/heptio/ark)
* Code of conduct

## Top priorities
* Add retries around setting up SSH port forwarding (it should make sugarkube abort if no port forwarding can be created - at the moment it always reports success). 
* Retry setting up SSH port forwarding in case the bastion hostname doesn't resolve

* The `kapps clean` command doesn't work - probably not merging in run units from the global config properly

* Update the cert manager and prometheus-operator kapps to delete CRDs when  deleted

* Setting 'versions' in stacks fails when there are 2 references to the same kapp (but different sources)
* Source URIs without branches should be ignored (unless an extra flag is set) to make it easy to ignore them in a stack by not setting a branch (it's safest to ignore them)

* Rerun with -race to try to find the cause of the intermittent concurrent map iteration and map write error when walking the DAG



* Find a way of stopping `kapp vars` or `kapps install --dry-run` failing if they refer to outputs from another kapp that don't exist (or if output can't be generated, eg jenkins when the cluster is offline). There is a `--skip-outputs` flag but it's easy to miss - perhaps add it to an error message?)

* Add flags to selectively skip/include running specific run steps (some steps - e.g. helm install - can be slow, which is annoying if you're debugging a later run step)
* kapps whose conditions are false won't be run even if they are explicitly selected with -i. That's annoying for `kapp vars`. Perhaps explicitly selected kapps should have their conditions overridden? Or add a flag for that?
* Add an '--only' option to the 'kapps' subcommands to only process marked nodes. Outputs will not be loaded for unmarked nodes/dependencies. This will speed up kapp development when you're iterating on a specific kapp and don't want to wait for terraform to load outputs for a kapp you don't care about. 
* There should be a flag to make sugarkube try to load generated outputs already on disk, but not actually execute the output steps (in case they take a long time)

* Fix issues around errors with actions:
  * it's safe to call 'create_cluster' multiple times, but calling 'delete_cluster' multiple times results in an error. Ideally we'd only throw an error on the first attempt and ignore it on subsequent ones (e.g. because we already successfully deleted the cluster this run)
  * running cluster_update twice for kops seems to kill ssh and make sugarkube lose connectivity. It dies with an error.

* It should be possible to set kapp vars that are maps and lists

### Cluster updates
* It should be easy to see what changes will be applied by kops - perhaps go to a two-stage approach with a '--yes' flag, to make a distinction between --dry-run and staging changes.

### Kapp output
* We also need to allow access to vars from other kapps. E.g. if one kapp sets a particular variable, 
  'vars' blocks for other kapps should be able to refer to them (e.g. myvar: "{{ .kapps.somekapp.var.thevar }}")
* Provide a 'varsTemplate' field to allow for templating before parsing vars. That'll help with things like reassigning
  a map. Template this block then parse it as yaml and merge it with the other vars (pretty sure templating & outputs make this obsolete).

### Developer experience
* Stream console output in real-time - see stern for an example of streaming logs from multiple processes in parallel. Add a flag to enable this.
* use ps (https://github.com/shirou/gopsutil/) to check whether SSH port forwarding is actually set up, and 
  if not set it up again. Also, when sugarkube is invoked throw an error if port forwarding is already set up
* Or do ssh using a golang library so we can make it more robust (reconnecting on dropped connections, etc)
* Improve the UX around using caches/workspaces, especially re updating while working on a change (sugarkube shouldn't bomb out but should update whatever it can)
* Make graph visualisations show kapp actions
* Allow graphs to be visualised in the install, delete or both directions (both looks pretty messy so it'd be good to have the option of each direction individually)
* Allow graph visualisations to show the individual run steps for each kapp (i.e. add a 'detailed' mode)
  
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
  
# Known issues
* It'd be nice to throw a more useful error if AWS creds have expired (e.g. for kops or trying to set up cluster connectivity) but it's currently out of our hands: really https://github.com/kubernetes/kops/issues/7393
