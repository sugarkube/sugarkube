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

## Cluster creation
Delegete to the appropriate provisioner:

### Minikube
* Make sure minikube is installed
* Check whether a local cluster is running with `minikube status`
* If that command fails, no cluster is running. If not, one is, i.e.
  `cluster_online={{ not minikube_status.failed }}`
* If a cluster is running, quit, unless a flag is set to reuse an existing
  cluster to avoid accidentally reconfiguring it (`ignore_existing=true`)
* If one isn't running, start with:
```
minikube start <values from the start_settings block in values.yaml>
```

