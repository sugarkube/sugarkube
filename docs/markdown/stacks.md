# Stacks
Stacks represent clusters, whether Kubernetes clusters or otherwise. They allow configuring broad settings that are more specific to clusters rather than to individual kapps.

Several settings are available to help you namespace resources so you can run multiple stacks in the same cloud account. E.g. if you need a hosted zone, name it something like `<cluster_name>.<profile>.<region>.example.com`. The same goes for other resources you create (e.g. load balancer names, S3 buckets, etc.). This is important to ensure each stack's resources are isolated from each other. See the sample project and notice how every resource uses variables to avoid naming conflicts between stacks.

Here are the available settings for a stack:

*	name - allows you to refer to a particular stack when there are multiple stacks defined in a stack YAML file. This is required.  
*	provider - the [provider](providers.md) to use to load configs from disk
*	provisioner - the [provisioner](provisioners.md) to use to create a cluster. Use `none` if you've got an existing cluster or don't want Sugarkube to create clusters          
*	account - the human-readable name of the account to run under, e.g. dev, prod, etc. This should be used to namespace your resources              
*	region - cloud provider region, e.g. eu-west-1. Not required when using the `local` provisioner               
*	profile - name of the profile to use. Profiles allow you to broadly configure different categories of clusters in a single cloud account. E.g. you may have a single account for testing, and want to use one profile with smaller instances for normal user acceptance testing, and larger instances for various different staging/performance testing clusters.               
*	cluster - name of the cluster. This should also be used to namespace your resources to avoid conflicts between multiple clusters running in the same cloud account.              
*	provider_vars_dirs - Directories that providers should search for config files
*	kapp_vars_dirs - Directories that should be searched for kapp variables
*	manifests - The list of manifests that should be applied to the stack
*	template_dirs - Directories to search for templates in if they aren't in a kapp

A good way to organise stack configs is to use YAML references to share common settings, and to group related stacks together. E.g. in a file called `aws-dev.yaml`, define multiple clusters using your dev account like this:
```
defaults: &defaults
  provider: aws
  provisioner: kops
  region: eu-west-2
  kapp_vars_dirs: some/path
  template_dirs: some/path

aws-dev1:
  <<: *defaults
  cluster: dev1
  manifests: 
  - manifest1.yaml
  - ...
```

# Overrides
