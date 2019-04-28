# To do
## Repo-related tasks
* CLA (see https://github.com/heptio/ark)
* Code of conduct

## Top priorities
* Update the 'kapp vars' command
* Fix passing a single flag to helm/tf where the file may not exist
* Support adding some regexes to resolve whether to throw an error if certain directories/outputs exist
  depending on e.g. the provider being used. Sometimes it doesn't make sense to fail if running a 
  kapp with the local provider because it hasn't e.g. written terraform output to a path that it 
  would do when running with AWS, etc. Some templates (e.g. terraform backends) should only be run for 
  remote providers, not the local one
* add subcommands for 'kapps clean' and 'kapps output' to run them across kapps
* Also add a --clean flag to make install/delete to clean kapps before running them
* Add a flag to install all dependencies for a kapp (i.e. mark all nodes in the subgraph)
  
### Merging kapp configs
* Create a 'validate' command to verify that binaries declared in `requires` blocks exist
* Support passing kapp vars on the command line when only one is selected

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
  invocations?

### Everything else
* Support declaring templates as 'sensitive' - they should be templated just-in-time then deleted (even on error/interrupts)

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

* Consider adding a cache so we can do cluster diffing to only install kapps that have changed to speed up
  deploying changes. Use a ClusterSOT for that.
* Create a cache manager whose job it is to organise where files are stored in a cache to enable the no-op provisioner to be used

## Other things to consider
* Is being focussed on clusters a mistake? 
    * We could help provision other hosted services, e.g. ElastiCache, BigQuery, etc. 
    * Maybe we should think more in terms of a 'context', e.g. dev1 could be used to run different related
      resources but with a different identifier to e.g. segregate a VPC from an ElastiCache or something (in 2 
      different stacks)
  