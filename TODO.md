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
  to enable this or keep the current behaviour
* Add '--jit-templating' (or make that the default?) so that kapps can have their templates written with dynamic content.

* Allow overriding manifest data from a stack file to permit, e.g. specifying the branch of a kapp at the stack level
  or extra vars, etc.
* Allow vars to be specified inside manifest files per kapp

* Emit a warning for kapps without a branch specified, but ignore them and proceed anyway
* Allow filtering kapps to apply/install
* Don't bomb out if there's no config file
* Don't always display usage if an error is thrown
* Implement deletion to tear down a stack
* Run automated scanning to check where errors aren't handled correctly
* Fix failing integration test

## Other things to consider
* Is being focussed on clusters a mistake? 
    * We could help provision other hosted services, e.g. ElastiCache, BigQuery, etc. 
    * Maybe we should think more in terms of a 'context', e.g. dev1 could be used to run different related
      resources but with a different identifier to e.g. segregate a VPC from an ElastiCache or something (in 2 
      different stacks)
  