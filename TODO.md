# To do
## Repo-related tasks
* CLA (see https://github.com/heptio/ark)
* Code of conduct

## Code-related tasks
* Support acquiring manifests with the acquirers - this will help multi-team setups, where the platform team can 
  maintain the main stack config, pulling in manifests from repos the app teams have access to (so they don't need
  access to the main config repo). Manifest variables will simplify passing env vars to all kapps in the manifest
  (e.g. for the tiller-namespace, etc.)
  
* Print important info instead of logging it
* Add support for verifying signed tags
* More tests 
* See if we can suppress warning in overridden makefiles by using the technique
  by mpb [described here](https://stackoverflow.com/questions/11958626/make-file-warning-overriding-commands-for-target)

* Need a way of dynamically adding variables to the databag. Perhaps if kapps write JSON to a file it could be 
  merged in? Then it'd be easy for users to control the frequency it runs. This is required to get the KMS key
  ARN which can then be templated into kapps. For now we can hardcode values because templating was expected
  to happen when creating a cache.
  We could have kapps declare the name of a JSON file in their sugarkube.yaml file that should be merged with 
  vars to allow them to dynamically update kapp vars. Or they could specify that stdout should be used, etc.

* Print out the plan before executing it
* Print details of kapps being executed
* Don't always display usage if an error is thrown
* Implement deletion to tear down a stack
* Fix failing integration test
* Wordpress site 2 isn't cached when running 'cache create' (probably due to it referring to a non-existent branch - 
  we should throw an error and abort in that case)

* Support variables at the manifest level so we can set manifest-wide vars. Use them as defaults for kapp vars (or 
  just namespace them under "manifest.vars" - decide which is better. The first would require we manually set a 
  default to the manifest vars ). E.g. this could be helpful for setting manifest-wide tiller-namespaces, etc.
* The `requires` block in `sugarkube.yaml` is currenntly useless. We should do several things with it:
  * Create a 'validate' command to verify that the necessary binary exists
  * Allow each value to have a corresponding config in the sugarkube-conf.yaml file that determines:
    * Default env vars (e.g. a kapp using helm should always take the TILLER_NAMESPACE env var from a certain
      place, one using kubectl will always need a NAMESPACE from somewhere, etc.). This will obviate the need to keep
      passing the same env vars to similar kapps
      
* Add support for config phases to provisioners. E.g. we might bring up a private kops cluster with a bastion, 
  install some stuff into it, but then want to scale down the bastion IG. That'll require 2 different kops configs
  so we should acknowledge they're for different phases of the lifecycle. Similarly to install new stuff into that 
  cluster we may need to relaunch the bastion, install stuff then remove it again.

* Stream console output in real-time
* Get rid of the duplication of mapping variables - we currently do it once in sugarkube.yaml files then
  again in makefiles. Try to automate the mapping in makefiles
* Create a single kapp descriptor and allow it to be used in stacks, manifests & kapps. Use the appropriate
  base directory for relative paths depending on where it's specified

* Fix passing a single flag to helm/tf where the file may not exist
* Support declaring templates as 'sensitive' - they should be templated just-in-time then deleted

* Support adding YAML/JSON/text output from kapps to the registry under e.g. 'output/<kappId>' where the 
  kapp ID can be just the name inside a given manifest, or a fully qualified ID if being used across manifests
* If a kapp uses output from an earlier kapp and it hasn't been run, throw an error
  
* Support taking the 'startAt' and 'runUntil' settings in the config file, so e.g. users can by default 
  start applying kapps from the point where their cluster is set up, but they can still explicitly set that
  flag to start at the start.

## Other things to consider
* Is being focussed on clusters a mistake? 
    * We could help provision other hosted services, e.g. ElastiCache, BigQuery, etc. 
    * Maybe we should think more in terms of a 'context', e.g. dev1 could be used to run different related
      resources but with a different identifier to e.g. segregate a VPC from an ElastiCache or something (in 2 
      different stacks)
  