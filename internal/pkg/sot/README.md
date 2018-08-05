# Sources of Truth
These determine which kapps are already installed in a cluster. We currently 
only have a `helm` SOT, but could in future support e.g. Consul, etcd, etc. if 
we convert these to plugins. [Viper](https://github.com/spf13/viper#remote-keyvalue-store-support) 
supports quite a few backends so could come in handy for this sort of thing.
