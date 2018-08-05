# Pseudocode for cluster creation
Describes how creating a cluster works at a high level.

## Invocation
* Invoke with different args depending on the provider. 

If AWS we need:
* the provider name
* account name
* profile name
* cluster name
* region

If minikube:
* provider name
* profile name
* cluster name

To do: Find some abstraction to allow creating friendly names for these to
reduce the number of required args, e.g. `dev1` could be loaded from
a config that contains:
```
aws_dev1:
  provider: aws
  account: dev
  profile: dev
  cluster: dev1
  region: eu-west-1   # could optionally define here or supply on the CLI

gcp_dev1:
  ...
```

These named configs would need unique names obviously.

## Config loading
Load all `values.yaml` files from the root of the `providers` directory
to the target cluster's region, merging values along the way. The root path
to  the `providers` directory should come from config or on the CLI.

## Prelaunch phase
Run any prelaunch kapps. Since these are kapps just go through the usual 
kapp installation process.

## Cluster creation

Delegete to the appropriate provisioner:

### Minikube
* Make sure minikube is installed
* Check whether a local cluster is running with `minikube status`
* If that command fails, no cluster is running. If not, one is, i.e.
  `cluster_online={{ not minikube_status.failed }}`
* If a cluster is running, quit, unless a flag is set to reuse an existing
  cluster to avoid accidentally reconfiguring it (`abort_if_cluster_exists=false`)
* If one isn't running, start with:
```
minikube start <values from the start_settings block in values.yaml>
```

### Kops
* Populate any default generated values, e.g. state store, cluster name, etc.
* Check to see if the cluster is online first. If it is and `abort_if_cluster_exists=true`
  then abort.
* Create the cluster with `kops create cluster`
  * Find a way to allow the parameters to be configurable. E.g. boolean 
    parameters like `--bastion` and `--encrypt-etcd-storage`. Also work out
    how to bridge generated/default values vs ones overridden in config.
* Once the cluster has been created, download the cluster config to a temp
  path: `kops get cluster --name {{ cluster_name }} --state={{ state }} -o yaml > {{ temp_path }}`
* Merge in any `kops` settings loaded from `values.yaml` files, and also the 
  network profiles. Then write it back to another temp file.
* The creation timestamp on the exported file isn't accepted when replacing the 
  config, so massage it: 
```
regexp: "(creationTimestamp: )(\\d{4}-\\d{2}-\\d{2}) (\\d{2}:\\d{2}:\\d{2})$"
replace: "\\1 \\2T\\3Z"
```
* Should we offer a confirmation before applying changes?
* Replace the kops config with: 
```
kops replace --name {{ cluster_name }} --state {{ state }} -f {{ updated_config_path }}
```
* Apply changes with:
```
kops update cluster --name {{ cluster_name }} --state {{ state }} --yes
```

### All provisioners - wait until the clusters are ready
* Wait until the cluster comes online. Poll `kubectl --context {{ kube_context }} get namespace`.
* Sleep for 30 seconds to let pods start to be installed
* Poll for all pods to be running. The following will return no stdout and a return code of 1 when the cluster is ready:
```
kubectl --context {{ kube_context }} -n kube-system get pod -o go-template='{{ '{{' }}range .items}}{{ '{{' }} printf "%s\n" .status.phase }}{{ '{{' }} end }}' ~ grep -V Running
```

## Post launch actions
