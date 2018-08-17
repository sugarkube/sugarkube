# Pseudocode for installing kaps
Describes how installing kapps works at a high level.

## Manifest loading
* Load all manifests from either the `stack` or given on the CLI.

## Planning
* Build a plan:
  * Build the lists of kapps to install and destroy based on the kapps in the 
    given manifests, and any CLI args (e.g. `filter` which allows selecting a 
    subset of kapps).
  * Use the configured `Source of Truth` to find out what's already installed 
    in the target cluster.
  * Refresh (or build) the cache (see below)
  * Read `credentials.yaml` from each kapp so we know what creds each one needs
  * Write the plan as YAML to the root of the kapp cache (by default 
    `_generated_plan.yaml`, or to a supplied path).

Also support loading a previously generated plan instead of always having to 
go through this process.

### Where to declare which creds a kapp needs?
If we store `credentials.yaml` we need to build/refresh the cache while 
generating the plan. If they're somewhere else we won't need to. What's 
better? It's probably clearer to store them in kapps, then people can configure
their CD pipelines to throw warnings if someone uses a credential †hat doesn't
contain the kapp's name (e.g. the wordpress kapp should probably only need
creds prefixed with `wordpress`).

## Refreshing the cache
Sometimes we'll want to build a cache for all kapps in all manifests for a 
target cluster, sometimes we'll only want to create one for the kapps to 
install and destroy. 

Caches can be built in named directories so you can maintain a cache for each
of your clusters. Refreshing a cache allows you to easily update your cache
when the manifest changes (e.g. by someone else in the team). It uses git 
sparse checkouts to support storing your kapps in a single repo, but building
your cache from kapps at different versions of that repo. 

* If no cache dir is given as an option, create a new temp dir for the cache.
* If a path to an existing cache dir is given, and `refresh_cache=true`, we 
  can proceed. Otherwise abort (configurably) so we don't blat a user's working 
  dir by mistake.
* Do a sparse git checkout of each kapp to install, along with all required
  sources. 
  * If we can't directly clone into the necessary output path, use symlinks
    to create the right relative dir structure so relative paths work 
    between sources.

## Applying changes
### Install kapps
For each kapp to install from the plan (and for the `helm` installer) do the 
following. Run each kapp in parallel, except ones with a value of 
`parallel: false` in the manifest:
* Find the dir in the cache containing `Chart.yaml`
* Expect the Makefile to be in the same dir
* Search for `values.yaml`, and `values-<profile/cluster>.yaml` (but 
  these could be in a sibling directory... how do we find them? Maybe have
  some key under `sources` in the manifest to call out where they are when 
  they're not in the kapp itself).
* Template/generate any values.yaml files into the same location
* Find the terraform directory (if there is one)
* Generate terraform files (backend and any others)
* Search for terraform vars files specific to the profile and/or cluster as 
  well as defaults
* Run `make all` passing things specific to the installer and cloud provider
  (it'd be good if the params can be standardised though so the same kapp can 
  be used with different providers, e.g. minikube, aws, gcp). Set `APPROVED` to 
  the value passed in on the CLI. 
  * For AWS/Helm pass the following env vars:
    * KUBE_CONTEXT
    * NAMESPACE
    * RELEASE
    * CLUSTER_PROFILE
    * CLUSTER
    * REGION
    * APPROVED
  * For the safest pipelines, run first with `APPROVED=false`, collect the 
    log output and check to see if terraform plans to destroy any infra. If
    so, wait for manual approval then rerun with `APPROVED=true` and the 
    same plan to apply the plan.
  * Alternatively, run with `--require-approval=false` in which case sugarkube
    will immediately run the task with `APPROVED=true` after first running 
    with `APPROVED=false` (to generate terraform plans, etc)
* Log all stdout to one file and all stderr to another file.
* By default, abort the kapp if anything was written on stderr (configurable 
  to ignore this, either globally or per kapp?)

Regarding running in parallel, the logic is:
1. Iterate through the kapps slated for installation
2. Add them all to a queue until any are seen that have `parallel: false`
3. Execute all in the queue in parallel
4. Run the kapp with `parallel: false` on its own
5. Return to 1 to process the full list.

This allows fanning out and back in again.

### Destroy kapps 
Do the above but in reverse. I.e.:
* Get the list of all kapps
* Filter out the ones that don't exist according to the SOT
* Destroy them in reverse, applying the same rules around parallelisation
