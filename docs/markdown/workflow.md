# Workflow
This outlines a sample workflow using Sugarkube to spin up and tear down Kubernetes clusters. 

## Set up
1. Define your [stack](stacks.md)
1. Create some manifests
1. Create some [kapps](kapps.md) or use [ours](https://github.com/sugarkube/kapps)

## Dev workflow
1. Create a [cache](cache.md) for your target cluster with `cache create`.
1. If you don't use the `cluster_update` [action](actions.md) to create a cluster, explicitly create a cluster by running `cluster create`. Don't do this if one of your kapps/manifests calls that action though. 
1. Run `kapps install` to install your kapps (and create your cluster if they call the `cluster_update` action)
1. Do your work - by editing/creating new kapps as necessary, and running `kapps install -i <manifest:kapp-id>` to reinstall just the kapp you're working on.
1. Tag your new kapps and update your configs to deploy the new kapp to a target cluster.
1. Tear down your dev cluster with `kapps delete`. 
1. If no kapp declares the `cluster_delete` [action](actions.md), manually run `cluster delete`.

Other useful commands while you're developing a kapp are:

* `cluster vars` - see all the interpolated, merged variables for your target cluster
* `kapp vars` - see the output of `cluster vars` plus the values of variables that will be supplied to your kapp and the values of any available outputs
* `kapp template` - render templates declared in your kapp
* `kapp clean` - run `make clean` across all your kapps to reset their state
