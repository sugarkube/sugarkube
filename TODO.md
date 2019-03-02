# To do
## Repo-related tasks
* CLA (see https://github.com/heptio/ark)
* Code of conduct

## Code-related tasks
* Support acquiring manifests with the acquirers - would this really help?
* Print important info instead of logging it
* Add support for verifying signed tags
* More tests 
* See if we can suppress warning in overridden makefiles by using the technique
  by mpb [described here](https://stackoverflow.com/questions/11958626/make-file-warning-overriding-commands-for-target)

* Need a way of dynamically adding variables to the databag. Perhaps if kapps write JSON to a file it could be 
  merged in? Then it'd be easy for users to control the frequency it runs. This is required to get the KMS key
  ARN which can then be templated into kapps. For now we can hardcode values because templating was expected
  to happen when creating a cache, but it would be more flexible to allow kapps to be templated just-in-time
  with values supplied dynamically. We could add a flag to kapps.install: '--jit-templating' and 
  to enable this or keep the current behaviour. Variables are merged just before kapps are installed, so just 
  updating variables in various sources (e.g. 'Manifest.Overrides') should be enough to implement this behaviour. 
  We could have kapps declare the name of a JSON file in their sugarkube.yaml file that should be merged with 
  vars to allow them to dynamically update kapp vars. Or they could specify that stdout should be used, etc.
* Add an action (defined in a kapp's sugarkube.yaml file) to indicate the cluster should be updated. This could run
  after adding additional variables dynamically.

* Remove init manifests - manifests should be idempotent. If one should only run while bootstrapping it should
  perform checks itself to avoid running multiple times

* Don't always display usage if an error is thrown
* Implement deletion to tear down a stack
* Fix failing integration test

* Fork go-yaml, set a large value for `emitter.best_width` in emitterc.go to much larger than the default 80. 
  Depend on it instead (see https://stackoverflow.com/questions/49475290/go-dep-and-forks-of-libraries)

## Other things to consider
* Is being focussed on clusters a mistake? 
    * We could help provision other hosted services, e.g. ElastiCache, BigQuery, etc. 
    * Maybe we should think more in terms of a 'context', e.g. dev1 could be used to run different related
      resources but with a different identifier to e.g. segregate a VPC from an ElastiCache or something (in 2 
      different stacks)
  