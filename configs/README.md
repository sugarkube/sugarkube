Note: The simplest way to implement config loading is to:

1. Search the config directory for all yaml files
2. Merge all file with the same names into different configs

In future we could think of a way to allow merging in values from e.g.
consul, etcd, some other k/v store, etc.
