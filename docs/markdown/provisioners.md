# Provisioners

When you're configuring a stack you need to declare the provisioner to use. Currently supported choices are:

* kops
* minikube
* none

Provisioners are responsible for creating and deleting clusters. More will be added in future.

YAML for provisioners must be placed under the `provisioner` key. Each provisioner supports different options as explained below.

## Kops

Kops supports creating clusters using a public or private topology. Ones created with a private topology use an internal load balancer for the API server and a private hosted zone for DNS records to it. It supports creating a bastion as a jump box to gain access to the VPC the cluster is created in. Creating clusters using the private topology is undoubtedly safer since the API server isn't exposed to the Internet by default. 

Sugarkube provides first-class support for private Kops clusters by setting up SSH port forwarding between your local machine and the bastion. It will set up port forwarding under several circumstances:

* A new cluster is created with a private topology and a bastion
* You pass the `--connect` flag to commands that support it (e.g. `kapps install` or `kapps delete`).

### Options

```yaml
provisioner:
  binary - path to the kops binary if you want to pin to a specific version (optional)
  ssh_private_key - path to the private SSH key. Used to set up SSH port forwarding if required
  bastion_user - username to SSH to the bastion as when setting up SSH port forwarding
  local_port_forwarding_port - local port to use for SSH port forwarding

params:        # parameters for Kops command line options
   global - applied to all commands
   create_cluster - CLI args for `kops create cluster`
   delete_cluster - CLI args for `kops delete cluster`
   update_cluster - CLI args for `kops update cluster`
   get_clusters - CLI args for `kops get clusters`
   get_instance_groups - CLI args for `kops get instancegroups`
   rolling_update - CLI args for `kops rolling-update`
   replace - CLI args for `kops replace`
```

Values for `create_cluster`, `delete_cluster`, etc can be found by running e.g. `kops create cluster -h`. Remove the leading '--' and change hyphens to underscores. E.g. `--master-count=3` should be defined as `master_count: 3`.

Booleans can be specified (e.g. for the `bastion` option) by declaring a key without a value, e.g. `bastion:`    

### Minikube

```yaml
provisioner:
   binary - path to the minikube binary if you want to pin to a specific version (optional)
   params:
      global - applied to all commands
      start - CLI args for `minikube start`
      delete - CLI args for `minikube delete`
```

Values for `start`, `delete` can be found by running `minikube start -h`, removing the leading '--' and replacing hyphens with underscores. E.g. `--disk-size=20g` should be defined as `disk_size: 20g`.

### None

This is a no-op provisioner that doesn't do anything. Use it if you're not using Kubernetes or don't want to use Sugarkube to create clusters for you.
