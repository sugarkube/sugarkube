# Kapps

Kapps are the release artefacts in Sugarkube. If you're using Helm charts, there'll be pretty much a 1:1 mapping between Helm charts and kapps. The difference is that kapps can also contain other things like scripts or terraform configs. Depending on the [installer](installer.md) there may also need to be other files, e.g. a Makefile. 

Kapps are organised into [manifests](manifests.md) which are further grouped into [stacks](stacks.md) that represent your actual clusters.

## Where kapps are configured

Kapps can be configured at multiple levels in a Sugarkube project (in order of precedence from lowest to highest):

* Global defaults (the lowest level of precedence) can be set in the project's `sugarkube-conf.yaml` file. A [default one](https://github.com/sugarkube/sugarkube/blob/master/sugarkube-conf.yaml) is available that configures dynamically searching for Helm/Terraform values/.tfvars files
* Default values can be set in the kapp's `sugarkube.yaml` file 
* Each manifest file that uses a kapp can declare defaults in its  `defaults` block, or be parameterised at the point the kapp is declared in the manifest (this has higher precedence than the manifest's `defaults` block)
* In a stack's `overrides` block

## Configuration

The following settings can be used to configure kapps everywhere it's possible to configure them (but some only make sense in certain places):

* id
* sources
* outputs
* state
* version
* args
* requires
* templates
* vars
* env_vars
* post_install_actions
* post_delete_actions
* pre_install_actions
* pre_delete_actions
* depends_on
* ignore_global_defaults

Sources are defined as a list of:

* uri - URI to the git repo of the form `<repo>//<path>#<tag>`, e.g. `git@github.com:sugarkube/kapps.git//incubator/wordpress#1.0.0`. Note - if your kapp is in the root of your repo, use `/` as the path, e.g. `git@github.com:example/kapps.git//#1.2.3`
* id - optional. If not set, the basename of the manifest file (i.e. the name without `.yaml`) will be used

Outputs are defined as a list of:

* id - this must be unique to the kapp
* format - one of `yaml`, `json` or `text`
* path - path to the local file that output will be written to
* sensitive - optional. If `true`, the output file will be deleted as soon as it's been read

Templates are defined as a list of:

* source - path to the source template. The path will be searched for first in the kapp (relative to the directory containing the kapp's `sugarkube.yaml` file), then in any directories configured in the stack's `kapp_vars_dirs` setting
* dest - the path to write the templated file to, relative to the kapp's `sugarkube.yaml` file

## Execution

When Sugarkube is executed, it:

1. Reads config files (which define your clusters, e.g. Kops on AWS or local Minikube, and the versions of which kapps to install into each cluster)
2. Clones the relevant git repos containing your kapps at the specified version
3. Invokes `Make` on them passing various environment variables. The Makefiles tailor exactly what they do based on these environment variables.

Most operations are run in parallel for speed (although that's configurable). This include cloning git repos and installing kapps.

The Makefile in each kapp acts as the interface between Sugarkube and exactly what a kapp does. Since our example kapps all use Helm + Terraform, we provide a set of default Makefiles that will:

* Lint Helm charts before installing them
* Initialise and execute Terraform code if a directory called `terraform_<provider>` exists

It's up to you to tailor each kapp's Makefile for your purposes. This approach makes kapps incredibly flexible and doesn't tie you into any particular programming language, tool (e.g. Helm/Terraform) or cloud provider. In future, we may even remove the dependency on Make, allowing you to invoke arbitrary scripts/binaries (e.g. if you'd rather write your releease scripts in your language of choice).
