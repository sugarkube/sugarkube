# Actions

Actions provide a way for [kapps](kapps.md) to callback to Sugarkube to manipulate its state or to make it do things. Supported actions are:

* cluster_update - instruct Sugarkube to update a cluster, similar to invoking the `cluster update` command. This allows you to create kapps that run before your cluster is created, e.g. to prepare the environment by creating hosted zones, a kops S3 state bucket, etc.
* cluster_delete - as above but for the `cluster delete` command.
* add_provider_vars_files - adds extra file paths to the list of paths that will be merged together by the provider. This allows you to write kapps that dynamically modify e.g. the kops cluster config. Read below for more on this 
* skip - will neither install nor delete the kapp

 Actions are specified as either pre or post install or delete, i.e.:

* pre_install_actions - run before the kapp is installed
* post_install_actions - run after the kapp is installed
* pre_delete_actions - run before the kapp is deleted
* post_delete_actions - run after the kapp is deleted

## Uses

Actions enable several advanced usecases such as:

* Creating a private VPC to install kops into, then triggering a `cluster_update` action to install kops into that VPC.
* Installing an OIDC provider (such as Keycloak) into the cluster, templating a new provider YAML file and adding it to the list of provider vars files. Then triggering a cluster update to reconfigure the cluster to use OIDC authentication instead of the default method.
* Specifying exactly at before or after which kapp a cluster should be created, updated or deleted when installing or deleting kapps.
