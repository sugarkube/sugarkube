# To do
## Repo-related tasks
* CLA (see https://github.com/heptio/ark)
* Code of conduct

## Code-related tasks
* add a 'kapps clean' task to run 'make clean' across kapps
* Also add a --clean flag to make install/delete to clean kapps before running them

### DAG algorithm
* When installing specific kapps, create a DAG for the entire set of manifests, then extract a subgraph for the target
  kapps. Now process the graph from the root: For all nodes which aren't the target nodes, if they declare output try 
  to load it from a previous run. If no previous output exists invoke `make output`, and only `make install/delete`
  on the target kapps, and execute any post actions they define.

### Merging kapp configs
* Create a 'validate' command to verify that binaries declared in `requires` blocks exists

* Support passing kapp vars on the command line when only one is selected

* Support adding some regexes to resolve whether to throw an error if certain directories/outputs exist
  depending on e.g. the provider being used. Sometimes it doesn't make sense to fail if running a 
  kapp with the local provider because it hasn't e.g. written terraform output to a path that it 
  would do when running with AWS, etc. 

### Kapp output
* We also need to allow access to vars from other kapps. E.g. if one kapp sets a particular variable, 
  'vars' blocks for other kapps should be able to refer to them (e.g. myvar: "{{ .kapps.somekapp.var.thevar }}")

* Provide a 'varsTemplate' field to allow for templating before parsing vars. That'll help with things like reassigning
  a map. Template this block then parse it as yaml and merge it with the other vars.

### Makefiles
* Get rid of the duplication of mapping variables - we currently do it once in sugarkube.yaml files then
  again in makefiles. Try to automate the mapping in makefiles
* Need to use 'override' with params in makefiles. How can we make that simpler?
* See if we can suppress warning in overridden makefiles by using the technique
  by mpb [described here](https://stackoverflow.com/questions/11958626/make-file-warning-overriding-commands-for-target)
* document  tf-params vs tf-opts and the same for helm in the makefiles

## Registry
* If kapp A writes output to the registry, and kapp B uses it, what happens if we try to delete kapp B? Since
  kapp A won't have been run, its outputs won't have been added to the registry. This may affect the ability to
  delete kapp B. There's a similar issue with only adding output to the registry after installing a kapp - it 
  stops us from planning later kapps.
* We should probably change the logic to opportunistically add outputs to the registry even while planning. If 
  that causes issues we could mark outputs as sensitive, but this'd potentially create unexpected behaviours re
  the freshness of outputs (imagine one that changes a value on each run).
* A better approach is probably to go with the unplannable kapp idea. When deleting kappB we could either
  reinstall kappA first (but what if it produces different output again?), or expect the value to be specified
  on the command line (but that would be cumbersome if deleting several kapps)...
* Actually I think a DAG is unavoidable. We'll need to have reversible and irreversible paths so we can
  e.g. always make something populate the registry. We should assume the output of a kapp is constant since
  they should be idempotent. That means we should be able to reload the output of a kapp from a file if it
  exists, or expect to regenerate it by reinstalling the kapp again (kappA) before installing/deleting
  kappB (and perhaps then deleting kappA as well).
* A usecase to think about is creating a shared RDS instance and needing to get the hostname from several kapps 

### Developer experience
* Standardise on camelCase or snake_case for config values
* Print important info instead of logging it. Print in different colours with an option to disable coloured output
* Print out the plan before executing it
* Print details of kapps being executed
* Stream console output in real-time
* use ps (https://github.com/shirou/gopsutil/) to check whether SSH port forwarding is actually set up, and 
  if not set it up again. Also, when sugarkube is invoked throw an error if port forwarding is already set up
  
### Config phases/states
* Enhance kapp selectors to allow specifying the first and last kapps to run. This'd make it easy to e.g. scale up
  the bastion ASG, install some stuff then scale it down again... but maybe users should just chain sugarkube 
  invocations.
* Support taking the 'startAt' and 'runUntil' settings in the config file, so e.g. users can by default 
  start applying kapps from the point where their cluster is set up, but they can still explicitly set that
  flag to start at the start.

### Everything else
* Fix passing a single flag to helm/tf where the file may not exist
* Support declaring templates as 'sensitive' - they should be templated just-in-time then deleted (even on error/interrupts)
* think about how to deal with downloading the kubeconfig file if the cluster has been configured to authenticate
  against keycloak - we should probably test whether the cluster is accessible already (i.e. if a vpn provides access
  into the cluster and the API server is accessible we won't necessarily need an SSH tunnel even if the API server
  is private)

* Support manifest sets and allow the level of parallelism between manifests to be configured. The default
  set will have a level of 1 so each manifest in the set will be executed separately, but other sets
  could allow any level of parallelism. This'll solve different app teams having their own manifests but
  allowing all of them to be installed simultaneously once the base cluster has been bootstrapped.

* Implement deleting clusters
  
* Support acquiring manifests with the acquirers - this will help multi-team setups, where the platform team can 
  maintain the main stack config, pulling in manifests from repos the app teams have access to (so they don't need
  access to the main config repo). Manifest variables will simplify passing env vars to all kapps in the manifest
  (e.g. for the tiller-namespace, etc.)

* Add support for verifying signed tags
* More tests 
* Fix failing integration test

* Update sample project
* Wordpress site 2 isn't cached when running 'cache create' (probably due to it referring to a non-existent branch - 
  we should throw an error and abort in that case)

* Create a cache manager whose job it is to organise where files are stored in a cache to enable the no-op provisioner to be used

## Other things to consider
* Is being focussed on clusters a mistake? 
    * We could help provision other hosted services, e.g. ElastiCache, BigQuery, etc. 
    * Maybe we should think more in terms of a 'context', e.g. dev1 could be used to run different related
      resources but with a different identifier to e.g. segregate a VPC from an ElastiCache or something (in 2 
      different stacks)
  