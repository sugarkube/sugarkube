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

To make it easier to work with multiple clusters in different states, Sugarkube provides a command to create a local cache of kapps declared for a cluster (the `cache create` command). This makes it easy to switch between working on kapps for different clusters.

## Creating a cache
Before you can run any commands on kapps you need to create a cache. This will checkout all sources for a kapp and group all kapps in each manifest together. Creating a cache can be done by using the `cache create` command. Running `cache create` on an existing cache will update it.

Sugarkube will clone git repos in parallel as far as possible to speed up creating a cache. It will also perform sparse checkouts to reduce the amount of data retrieved. Sugarkube will propagate any errors from git to prevent you losing uncommitted work (e.g. if running `cache create` again to update your cache).

If you browse the cache that's created you'll see how kapps are grouped by manifest and how symlinks are created between each source in a kapp.

## Scenario
Imagine you needed to add a new feature to wordpress from the above matrix. You create a cache for the dev1 cluster, go to the wordpress kapp and create a new branch (`br-new`). You develop and commit to your branch as usual. 

While you're doing this you get asked to urgently fix a bug in prod. So you create another cache for prod, hotfix the affected application in a new dev cluster and merge and tag the fix when you're happy. Next you update the staging stack's config to test your new kapp, and if you're happy update the prod one to release it.
 
 You can now go back to working in dev1. You could update your configs to pull your fixed kapp, run `cache create` again to pull in those changes, and pick up where you left off.
 