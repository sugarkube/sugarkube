# The cache
Sugarkube solves the problem of which version of each kapp to deploy to each target cluster. Complex environments (especially in enterprises) may have multiple live environments or need to respect change windows, etc. A team responsible for providing a Kubernetes platform for the rest of the organisation may need to treat a dev cluster as live since if it goes down it'd block the rest of the organisation's developers from working. 

We could list the versions of each kapp to install into each cluster in a matrix:

|           | dev    | staging | prod1 | prod2 |
|-----------|--------|---------|-------|-------|
| ingress   | 1.1.0  | 1.1.0   | 1.0.0 | 1.0.0 |
| memcached | 1.2.0  | 2.4.3   | 2.4.3 | 2.4.2 |
| wordpress | br-new | 3.2.1   | 3.2.0 | 3.2.0 |


By explicitly choosing the version of a kapp to install to a cluster you can control exactly when it gets deployed. Since your configs will be versioned then you also gain the ability to recreate the state of a cluster (excluding data) by checking out earlier versions of your configs and installing the kapps it defines.

To make these workflows more ergonomic Sugarkube provides a command to create a local cache of kapps declared for a cluster (the `cache create` command). This makes it easy to switch between working on kapps for different clusters. 