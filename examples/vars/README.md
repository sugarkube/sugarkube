# Variables
Sometimes you need to pass configuration to kapps that are related to your 
project, not the kapp itself. For example you may need to install two instances
of Jenkins, one for two different clients. The Jenkins kapp might set its 
ingress hostname to `jenkins.<DOMAIN_NAME>`, but the value of `DOMAIN_NAME` 
will change per client. Using vars files allows you to install the same kapp
multiple times parameterised differently, so you could install it once at 
`jenkins.client1.com` and once at `jenkins.client2.com`. 

## Directory structure
The files in this directory are hierarchically merged and passed through to kapps
during installation. Sugarkube will look for directories related to the target 
cluster, and will merge variables following the following rules:

* If there's only a subdirectory. merge values
* If there are multiple subdirectories, look for any called any of the 
  following (in this order) as defined in the stack:
  * provider
  * provisioner
  * profile
  * cluster
  * installer (default `make`)
  * manifest
  * `<KAPP_ID>.yaml`      
  
In the above, `KAPP_ID` could be `wordpress-site1.yaml` for example if there 
are multiple Wordpress instances.

E.g. so an example `stack.yaml` file that defines the following cluster:
```
local-standard:
  provider: local
  provisioner: minikube
  profile: local
  cluster: standard
```
Files will be searched for in subdirectories called any of the following at 
each level:
* local
* minikube
* standard

These are similar rules to how provisioner values are loaded, so you could also
create vars files in your provisioner directory structure. However keeping them
separate makes your provisioner directories more reusable since they're less
tightly coupled to each project you work on. 

## Usage in kapps
Kapps receive each value as environment variables, prefixed by the basename  
of the file containing the variable. The exception is the file `values.yaml`
which are passed without any prefix. All variable names are upper-cased and have
hyphens converted to underscores.

Here's an example that illustrates the variable merging logic for a kapp with
an ID of `wordpress`:
```
vars
|-- local
  |-- values.yaml   # defaults, passed without a prefix
  |-- wordpress.yaml   # keys will be prefixed with `WORDPRESS_
  |-- standard      # corresponds to the cluster name in the example stack.yaml file above
    |-- wordpress.yaml  # any values here will override values with the same 
                        # key in `local/wordpress.yaml` because files lower in 
                        # the hierarchy take precedence. 
```
If `local/wordpress.yaml` contains:
```
hosted-zone: example.local
```
and `local/standard/wordpress.yaml` contains:
```
hosted-zone: example.com
```
Then the `wordpress` kapp will be passed `WORDPRESS_HOSTED_ZONE=example.com` 
when being installed into the `standard` cluster. It's up to the kapp's 
`Makefile` to use this value e.g. as a parameter to Terraform or Helm.
