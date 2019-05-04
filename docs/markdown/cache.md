# Caches
Sugarkube solves the problem of which version of each kapp to deploy to each target cluster. Complex environments (especially in enterprises) may have multiple live environments or need to respect change windows, etc. A team responsible for providing a Kubernetes platform for the rest of the organisation may need to treat a dev cluster as live since if it goes down it'd block the rest of the organisation's developers from working. 

We could list the versions of each kapp to install into each cluster in a matrix:

|           | dev1   | staging | prod1 | prod2 |
|-----------|--------|---------|-------|-------|
| ingress   | 1.1.0  | 1.1.0   | 1.0.0 | 1.0.0 |
| memcached | 2.4.3  | 2.4.3   | 2.4.3 | 2.4.2 |
| wordpress | br-new | 3.2.1   | 3.2.0 | 3.2.0 |

The above shows 2 prod clusters running different versions of memcached - the team are waiting for an upcoming change window before pushing the updated memcached to prod2. Staging has some changes to both wordpress and the ingress controller being tested, while in dev1 a new branch of wordpress is being worked on.

By explicitly choosing the version of a kapp to install to a cluster you can control exactly when it gets deployed. There's no CD magic. And by versioning your configs you also gain the ability to recreate the state of a cluster (excluding data) by checking out an earlier version installing the kapps it defines.

To make it easier to work with multiple clusters in different states, Sugarkube provides a command to create a local cache of kapps declared for a cluster (the `cache create` command). This makes it easy to switch between working on kapps for different clusters as we'll explain. 

