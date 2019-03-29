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

* use ps (https://github.com/shirou/gopsutil/) to check whether SSH port forwarding is actually set up, and 
  if not set it up again. Also, when sugarkube is invoked throw an error if port forwarding is already set up

* document  tf-params vs tf-opts and the same for helm in the makefiles

* Print out the plan before executing it
* Print details of kapps being executed
* Don't always display usage if an error is thrown
* Implement deleting clusters
* Fix failing integration test
* Wordpress site 2 isn't cached when running 'cache create' (probably due to it referring to a non-existent branch - 
  we should throw an error and abort in that case)

* Support variables at the manifest level so we can set manifest-wide vars. Use them as defaults for kapp vars (or 
  just namespace them under "manifest.vars" - decide which is better. The first would require we manually set 
  defaults for manifest vars ). E.g. this could be helpful for setting manifest-wide tiller-namespaces, etc.
* The `requires` block in `sugarkube.yaml` is currently useless. We should do several things with it:
  * Create a 'validate' command to verify that the necessary binary exists
  * Allow each value to have a corresponding config in the sugarkube-conf.yaml file that determines:
    * Default env vars (e.g. a kapp using helm should always take the TILLER_NAMESPACE env var from a certain
      place, one using kubectl will always need a NAMESPACE from somewhere, etc.). This will obviate the need to keep
      passing the same env vars to similar kapps
      
* Add support for config phases to provisioners. E.g. we might bring up a private kops cluster with a bastion, 
  install some stuff into it, but then want to scale down the bastion IG. That'll require 2 different kops configs
  so we should acknowledge they're for different phases of the lifecycle. Similarly to install new stuff into that 
  cluster we may need to relaunch the bastion, install stuff then remove it again.
  * Maybe we need to support an extra key (e.g. region/account/cluster - 'config'?) to allow different named yaml files
    to be merged together for a target cluster. E.g. we could have yaml files called 'bastion.yaml' and 'no-bastion.yaml'
    which will contain kops fragments for scaling the bastion up and down respectively. Users could then first invoke
    sugarkube passing this extra flag (e.g. --extra-config=bastion) to make it patch the kops config to bring up the 
    bastion. Then they could apply their kapps cherry-picking where to start from, what to include/exclude, etc. via
    selectors. Finally they could run it again with e.g. --extra-config=no-bastion to have sugarkube patch the kops 
    config again with the YAML to scale down the bastion. Users would then just need to chain those commands together
    (or invoke them sequentially). Those extra configs may need to be merged in with the highest priority though (i.e.
    last) to be really useful and to actually override existing configs, but for our use case I think we'd be OK without
    that.
  * We could perhaps have extra CLI args to take a list of different extra configs to apply, e.g. 
    `--extra-configs bastion,,no-bastion` and sugarkube could be run first with the `bastion` extra config, then with
    no extra config, then again with the `no-bastion` config. This'd effectively chain the invocation for users and
    would prevent them from forgetting to e.g. tear down a bastion.
  * Actually a more versatile solution would be to use the existing idea of manifests and allow them to be configured
    with extra file basenames to search for and to merge with the highest priority. That'd mean these 'state manifests' 
    would
    * be able to define additional config for provisioners, etc
    * run (versioned) kapps to do whatever's necessary to generate that extra config, or to put the cluster into a 
      desired state
    With this it then just becomes a question of how to select the manifests to run for a stack - state manifests 
    shouldn't be included in the main stack's list of manifests, but rather be able to be "topped and tailed" into a 
    run, e.g. to put the cluster into a known state, run the manifests as usual (with whatever selectors are required)
    and then put the cluster into a different state. If we needed to go through a series of states we could invoke 
    Sugarkube multiple times but that might complicate populating the registry in certain instances. In that case there's
    no reason why a stack config couldn't include state manifests in order to transition the cluster. 
    We could allow states to be run by having extra flags for `--start-state` and `--end-state`

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

* We should probably merge structs using mergo.WithAppendSlice and mergo.WithOverride (e.g. 
   mergo.Merge(result, fragment, mergo.WithAppendSlice, mergo.WithOverride)) but whichever we do will cause
   problems for some people. We should probably make it a config option as to whether to enable WithAppendSlice 
   or not. 

## Other things to consider
* Is being focussed on clusters a mistake? 
    * We could help provision other hosted services, e.g. ElastiCache, BigQuery, etc. 
    * Maybe we should think more in terms of a 'context', e.g. dev1 could be used to run different related
      resources but with a different identifier to e.g. segregate a VPC from an ElastiCache or something (in 2 
      different stacks)
  