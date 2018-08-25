# Road Map
**0.1.0**:
* Launch a minikube cluster works
* Install kapps, but:
  * without templating/file generation
  * wihtout consulting a SOT so all kapps in a manifest will be 
    installed/destroyed regardless of the current state of the cluster
* Add licence (Apache 2 probably)  
  
**0.2.0**:
* Implement cache diffing
* Implement SOTs to enable cluster diffing
  
**0.3.0**:
* Template values.yaml files and terraform files from vars
* Create `backend.tf` files to allow terraform be backed by different backends
  * Don't assume S3. This should be configurable too.
* Implement cluster diffing so we can install only those kapps that need 
  installing and to destroy those that need removing

**0.4.0**:
* Add a kops provisioner
* Work on bootstrapping before running a provisioner
  * E.g. kops needs an S3 backend encrypted with KMS. How do we create that 
  before running kops, and where do we store the ARN, etc?
  
**0.5.0**:
* Implement a state store if necessary so that, e.g. KMS key ARNs can be stored 
  (although perhaps we don't need this and can just use aliases? What other 
  use cases are there?)
* Use acquirers to acquire manifests from git repos as well as local files 
* Catch up on tests

**0.6.0**:
User-friendliness/ergonomics:
* Add a default config file with the usual platform-dependent search paths
  (e.g. ~/.sugarkube, etc. (see os.UserCacheDir())).
* Add flags to the root command for:
  * setting the log level
  * specifying the path to a config file 
* Print the output of commands instead of only logging
  * Bear in mind in future we may want to allow different output formats, e.g. 
  yaml, json, etc., so don't just print to stdout.

**0.7.0**:
* Implement parameterisers and remove helm/terraform specific code in the 
  MakeInstaller:
  * Parse configs
  * Implement them
  * Support installerConfigs to disambiguate when multiple Makefiles are found
