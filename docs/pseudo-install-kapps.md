# Pseudocode for installing kaps
Describes how installing kapps works at a high level.

## Manifest loading
* Load all manifests from either the `cluster default` or given on the CLI.

## Kapp cache
* If no cache dir is given as an option, create a new temp dir for the cache.
* If a path to an existing cache dir is given, and `refresh_cache=true`, we 
  can proceed. Otherwise abort (configurably) so we don't blat a user's working 
  dir by mistake.
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

* Do a sparse git checkout of each kapp to install, along with all required
  sources. 
  * If we can't directly clone into the necessary output path, use symlinks
    to create the right relative dir structure so relative paths work 
    between sources.
