# Road Map
**0.1.0**:
* Launching a minikube cluster works
* Installing kapps works, but there's no templating/file generation
* Kapps are always installed. There's no consultation of a SOT to only install
  kapps that need installing based on what's already in the cluster

**0.2.0**:
* Add a default config file with the usual platform-dependent search paths
  (e.g. ~/.sugarkube, etc.).
* Add flags to the root command for:
  * setting the log level
  * specifying the path to a config file 
* Print the output of commands instead of only logging
  * Bear in mind in future we may want to allow different output formats, e.g. 
  yaml, json, etc., so don't just print to stdout.

**0.3.0**:
* Template values.yaml files and terraform files from vars
* Create `backend.tf` files to allow terraform be backed by different backends
  * Don't assume S3. This should be configurable too.
* Implement SOTs to only install kapps that need installing and to delete ones
  that need deleting

**0.4.0**:
* Add a kops provisioner
* Work on bootstrapping before running a provisioner
  * E.g. kops needs an S3 backend encrypted with KMS. How do we create that 
  before running kops?

**0.5.0**:
* Implement a state store so that, e.g. KMS key ARNs can be stored (although 
  perhaps we can just use aliases?)
* Use acquirers to acquire manifests to support git as well as local manifests 