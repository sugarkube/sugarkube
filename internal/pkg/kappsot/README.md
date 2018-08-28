# Sources of Truth
These determine which kapps are already installed in a cluster. We currently 
only have a `helm` kapp SOT, but could in future support e.g. Consul, etcd, etc. if 
we convert these to plugins. [Viper](https://github.com/spf13/viper#remote-keyvalue-store-support) 
supports quite a few backends so could come in handy for this sort of thing.

We could have made kapps implement a target to tell us whether they're already
installed, but that could potentially lead to a lot of duplication. Also, it
could make it complicated for kapps to authenticate with backends like Consul,
so this feels like an activity that should be done centrally by Sugarkube.  
