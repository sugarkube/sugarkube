# Road Map 
* Try to get rid of makefiles
  - consider adding 'run units' to sugarkube.yaml files instead. Most makefiles only really execute one or two lines of bash, but there are a load of conditionals for whether approved=true/false, whether files exists, etc. This could all be moved out to a few lists of run units, e.g.:
  
```
units:
  init:
  - terraform init
  plan:
  - terraform plan -var HOST={{ .kapp.vars.host }}              # these are all templates so vars can be interpolated
  - helm validate
  apply:
  - terraform apply
  - helm install
```
Standard units could be put in the global config. If users want to do more complicated things they could call their own script instead. We could write all variables to a yaml/json file instead of exporting a ton of env vars, and a user's script could read those instead.

After running each unit, sugarkube should try to load any configured outputs and retemplate files. That'd provide an easy way to use e.g. resources created by terraform in helm charts, etc. Sugarkube could generate a simple script per unit, or generate a temporary makefile. That'd make it simple for users to rerun units directly (i.e. not through sugarkube). 

Ultimately this approach should hopefully mean we could get rid of our common makefiles altogether.
