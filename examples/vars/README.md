# Variables
The files in this directory are hierarchically merged and pass through to kapps
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

E.g. so if the `stack.yaml` file defines the following cluster:
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

Kapps receive each value as environment variables, prefixed by the basename of 
of the file containing the variable. The exception is the file `values.yaml`
which are passed without any prefix. All variable names are upper-cased and have
hyphens converted to underscores.
