# Kapps

## Execution
When Sugarkube is executed, it:

1. Reads config files (which define your clusters, e.g. Kops on AWS or local Minikube, and the versions of which kapps to install into each cluster) 
1. Clones the relevant git repos containing your kapps at the specified version
1. Invokes `Make` on them passing various environment variables. The Makefiles tailor exactly what they do based on these environment variables. 

Most operations are run in parallel for speed (although that's configurable). This include cloning git repos and installing kapps.

The Makefile in each kapp acts as the interface between Sugarkube and exactly what a kapp does. Since our example kapps all use Helm + Terraform, we provide a set of default Makefiles that will:

* Lint Helm charts before installing them
* Initialise and execute Terraform code if a directory called `terraform_<provider>` exists

It's up to you to tailor each kapp's Makefile for your purposes. This approach makes kapps incredibly flexible and doesn't tie you into any particular programming language, tool (e.g. Helm/Terraform) or cloud provider. In future, we may even remove the dependency on Make, allowing you to invoke arbitrary scripts/binaries (e.g. if you'd rather write your releease scripts in your language of choice).