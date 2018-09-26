# Variables
Sometimes you need to pass configuration to kapps that are related to your 
project, not the kapp itself. For example you may need to install two instances
of Jenkins, one for two different clients. The Jenkins kapp might set its 
ingress hostname to `jenkins.<DOMAIN_NAME>`, but the value of `DOMAIN_NAME` 
will change per client. Using vars files allows you to install the same kapp
multiple times parameterised differently, so you could install it once at 
`jenkins.client1.com` and once at `jenkins.client2.com`. 

## Directory structure
The YAML files in this directory are hierarchically merged and the resulting 
dictionary/map can be used to template files for kapps. Sugarkube will merge 
files hierarchically (that is, from the root to leaves, with files closer to 
leaves overriding values higher up) by searching for directories with any 
of the following names (with values coming from the configured stack): 

  * provider
  * provisioner
  * account (if relevant to the provider)
  * region (if relevant to the provider)
  * profile
  * cluster
  * installer (default `make`)
  * manifest
  * `<KAPP_ID>.yaml`      

Note that the above will be searched for in every subdirectory. However, 
subdirectories that don't match any of the above will be ignored and won't be 
searched further. So unlike with providers where there's a defined hierarchy, 
variable file directories are more free-form. This means if you could define 
some rather strange directory structures which may not behave as expected. 
However, it gives more flexibility because it means variables files aren't tied 
to a particular provider. If you want to do that, put your variables into the 
providers directory tree.

In the above, `KAPP_ID` could be `wordpress-site1.yaml` for example if there 
are multiple Wordpress instances.

E.g. so an example `stack.yaml` file that defines the following cluster:
```
local-standard:
  provider: local
  provisioner: minikube
  profile: local-mini
  cluster: standard
```
Files will be searched for in subdirectories called any of the following at 
each level:
* local
* minikube
* local-mini
* standard

These are similar rules to how provisioner values are loaded, so you could also
create vars files in your provisioner directory structure. However keeping them
separate makes your provisioner directories more reusable since they're less
tightly coupled to each project you work on. 

## Usage in kapps
Kapps can declare how to receive variables. They can either be used to template
files, or can be passed as environment variables.

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
Then if the `wordpress` kapp used the variable `hosted-zone` it would have the 
values `example.com`. It could use this value to template a file, or opt to
have it passed as an environment variable under an arbitrary key. 
