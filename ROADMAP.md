# Road Map 
**0.6.0**:
* Implement a state store if necessary so that, e.g. KMS key ARNs can be stored 
  (although perhaps we don't need this and can just use aliases? What other 
  use cases are there?)
* Decide whether we need to support passing the output of one kapp to another
  (e.g. a shared RDS DB hostname)
  
**0.7.0**:
* Implement cache diffing
* Implement SOTs to enable cluster diffing
* Implement cluster diffing so we can install only those kapps that need 
  installing and to destroy those that need removing
* Catch up on tests

**0.8.0**:
* Work on bootstrapping before running a provisioner
  * E.g. kops needs an S3 backend encrypted with KMS. How do we create that 
  before running kops, and where do we store the ARN, etc?
* Use acquirers to acquire manifests/vars files from git repos as well as local files 

**0.9.0**:
* Disambiguate when multiple Makefiles, etc. are found
