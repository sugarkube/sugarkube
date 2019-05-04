# Pseudocode for installing kaps
Describes how installing kapps works at a high level.

## Manifest loading
* Load all manifests from either the `stack` or given on the CLI.

## Diff the cluster
* Diff the state of the cluster against all the kapps in the manifests:
  * Build the lists of kapps to install and destroy based on the kapps in the 
    given manifests, and any CLI args (e.g. `filter` which allows selecting a 
    subset of kapps).
  * Use the configured `Source of Truth` to find out what's already installed 
    in the target cluster.
  * Refresh (or build) the cache (see below)
  * Read `sugarkube.yaml` from each kapp so we know what creds each one needs
  * Print the diff as YAML on stdout, or to a file (default 
    `<cache_dir>/_generated_diff.yaml`)

Also support loading a previously generated diff instead of always having to 
go through this process.

### Where to declare which secrets a kapp needs?
Kapps can include a `sugarkube.yaml` file which will be outputted verbatim by
the `cluster diff` command. Thsi can be used by CI/CD systems to discover which
secrets a kapp needs making available during installation as environment 
variables.

The `sugarkube.yaml` file can also contain additional metadata about the kapp
such as what type of kapp it is. While the types of some kapps can be identified
via heuristics (e.g. a kapp includes a Helm chart if it includes a file called
`Chart.yaml`), some can't be so easily so must be explicitly specified. They
can be specified in `sugarkube.yaml`.

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
Process each manifest sequentially, processing all kapps in it in parallel. For
`helm` charts, do the following:
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
* Run `make install` passing things specific to the installer and cloud provider
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

All kapps in a manifest are run in parallel, and processing of all kapps in the
manifest must finish before Sugarkube will move on to the next manifest. This
makes it simple to control fanning out and in again by just creating multiple
manifests. E.g. if you want kapps A, B & C to run in parallel, D to be run 
after all of them, and then E & F to be run again in parallel, create 3 
manifests, one for A, B & C, the next just for D, and then one more for E & F.

### Destroy kapps 
Do the above but in reverse. I.e.:
* Get the list of all kapps
* Filter out the ones that don't exist according to the SOT
* Destroy them in reverse, applying the same rules around parallelisation
