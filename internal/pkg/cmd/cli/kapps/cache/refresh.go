package cache

// Refreshes an existing kapps cache. This could perhaps be merged into a single
// `cache` command with a flag `--refresh` or `--update` to run in an existing
// cache directory. I'm not sure we need 2 separate commands that are so
// similar.

// Refreshing means:
// * Read all the kapps from the manifests
// * Do git sparse checkouts and build the cache
// * Add flags for dealing with edited kapps (ignore, abort, etc.) and filtering
//   kapps vs just checking them all out.
