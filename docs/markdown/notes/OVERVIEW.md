# Sugarkube - Kubernetes Route-to-Live
Sugarkube brings the simplicity of something like a python `requirements.txt` file, npm `package.json` or Java `pom.xml` to Kubernetes clusters. You create `kapps` (Kuberenetes apps) which are versioned artifacts containing a Makefile with some standard targets. They know how to bootstrap themselves and to create any necessary infrastructure (e.g. S3 buckets, RDS databases, etc.). Sugarkube will then deploy the versions you specify onto your target clusters, creating additional infrastructure as necessary.

By following the best practices we've developed, you'll be able to have all of the following with minimal effort:

* Fully provisioned, per-developer ephemeral clusters, either local or remote
* Ephemeral test clusters
* Robust disaster recovery (we can't help with data at this point though)
* Multiple clusters per AWS/Google Cloud/whatever account
* Versioned infrastructure kept in lockstep with the Helm charts that need it

Sugarkube is the Kubernetes deployment process you'd probably develop if you had time. It's incredibly flexible because it's a combination of architectural best-practices, and a binary to use them. It's flexible enough to be used with existing infrastructure and is ideal for greenfield projects. At it's core it's a set of best-practices and versioning using Make as an interface, so in fact it isn't even restricted to K8s or Helm charts. It's flexible, open and powerful.

Focus on building your product, not your deployment pipeline. Join us to help create a generic, flexible, secure deployment process that works in the real world for most companies.

## Features/Summary

* Kapps - Standalone, versionable deployment artifacts.
  * Plugins can fetch kapps from several backends, initially just from git repos but in future backends like S3, chart museum, artifactory, etc.
  * Kapps can specify dependencies to allow you to pull different requirements from different locations. E.g. you might want to version your Helm `values.yaml` files separately to your kapps, or pull in some shared makefile targets, etc. 
  * Kapps know how to bootstrap themselves which makes it easy to install them into new clusters (both prod and non-prod) and perform disaster recovery. E.g. if they require SSL certificates they must provide scripts to generate them. These can be then be passed as parameters to e.g. Helm. 
  * They are responsible for creating any infrastructure that will be solely used by that kapp, e.g. S3 buckets, RDS databases, etc.
    * Shared infrastructure (i.e. used by more than a single kapp) should be deployed as a kapp as well. These kapps can be configured to run before other kapps are installed, and can be used to e.g. create shared load balancers, CloudFront distributions, etc.
  * Creating infrastructure first per kapp means that dynamic values, e.g. RDS hostnames, can be created by e.g. terraform, then exported as env vars and passed as values into Helm to configure the Helm chart.
  * Their only contract is they must implement 2 `make` targets: `all` and `destroy`:
    * `all` must create any necessary infrastructure when running against remote clusters, but shouldn't do much when targetting a local e.g. minikube cluster. It should also install e.g. the Helm chart into the cluster (but using Helm isn't a requirement).
    * `destroy` must delete e.g. the Helm chart from the cluster and also tear down any infrastructure created. If a `quick` env var is set, they might just delete the infrastructure since this indicates the cluster is about to be destroyed, so e.g. there's no point slowly deleting a Helm chart that uses a stateful set when the cluster itself is about to be terminated.
    * Environment variables are provided to the `make` targets so the implementations can decide what to do based on whether the target is a local or remote cluster, etc.
  * Since kapps use `make`, default sets of `make` targets are provided to simplify the creation of `Makefiles`, especically since most kapps will have the same default targets. The advantage is that any make target can be overridden per kapp for maximum flexibility.
  * Kapps should be written so that all infrastructure is namespaced to permit multiple clusters per AWS/Google/whatever account.
    * This can make more efficient use of accounts. A single dev account could be shared by multiple developers. If all infra resources are namespaced they won't clash with each other. E.g. S3 buckets should contain the cluster name, region, etc. Each cluster should have its own Hosted Zone, etc. These architectural principles will be documented, and in future a linter will be offered to ensure kapps conform.
    * Planning up-front for multiple clusters per cloud account, per region, etc. means you could bring up sister clusters for zero-downtime upgrades. It also allows you to bring up a test cluster in prod which, in an ideal world you wouldn't have to, but perhaps prod is a snowflake with certain IPs whitelisted with a third party, etc. This flexibility allows you to get things done when other aspects of your infrastructure aren't as flexible.
* Manifests - Similar to pip `requirements.txt` files, Java's `poms` and NPM's `package.json`, manifests describe which versions of which kapps to install on a target cluster. At their core they allow you to build and maintain clusters (either local or remote) that will have whichever versions of whichever kapps you want installed on them.
  * Multiple manifests can be created (e.g. one per cluster) to support multiple live environments (e.g. monitoring, dev/test/staging/prod). 
  * Manifests can be composed. So for example, a platform team could maintain one (or several) manifests to configure the core platform to e.g. install `cert-manager`, `kiam`, etc. Each dev teams can then maintain their own manifests to install their kapps at different versions into different clusters.
  * Initially manifests will be read from disk, but in future they could come from pluggable backends (e.g. consul).
* Cluster provisioners - Plugins that allow you to provision local clusters with minikube, or remote clusters with kops, EKS, GKE, etc.
* Secrets providers - Kapps declare which secrets they need from whichever secrets provider they want. This means some secrets could come from e.g. vault, or from the environment. If secrets come from the environment, they could either be generated per kapp when running against dev clusters, or be supplied by e.g. Jenkins when running as part of a CI/CD pipeline.
* Kapp caches - These are working directories that contain each target kapp in a manifest at a target version, along with any dependencies. 
  * You can have multiple kapp caches, one per cluster. This allows you to clearly see which kapps are installed at which version, and provides an easy way to develop new features. 
  * This concept is important because different versions of different kapps may be installed on different clusters. For example you may be running 1.0.0 of `cert-manager` in your live environment, while you're testing 1.1.0 in a lower env. If you have to then fast-track a fix for live for a different kapp, it's useful to have a local working directory tree that contains `cert-manager` at 1.0.0 instead of 1.1.0.
* The `sugarkube` binary itself provides the following:
  * A single command to launch a cluster and install kapps into it from whichever manifest(s) are supplied. Another command will tear it down. 
    * This allows you to have:
      * Per-developer local/remote k8s clusters
      * Ephemeral testing/staging/performance testing clusters that could be, for example, brought up in the morning, used throughout the day, then terminated at night.
      * Regular fully-automated disaster recovery drills (excluding data)
    * Creating a cluster begins by running a configurable initialisation step. This can be used to e.g. provision the infrastructure you'll install kops into, or create load balancers to feed into kops. It then invokes the configured cluster provisioner to bring a cluster online.
  * A command which reads manifests and idempotently installs kapps into the specified target cluster:
    * It can query cluster state via several plugins, by default by directly invoking the `Helm` binary. In future this could be through backends like Consul. After determining which kapps are already installed it will create a plan describing which kapps will be installed at which versions, and which credentials each needs from each secret provider. 
      * Sugarkube won't reinstall kapps that are already installed at the target versions. This makes installing kapps fast and idempotent.
    * The plan can be used by e.g. Jenkins to make required credentials available to the kapp at deployment time, as well as to provide a clear audit trail.
    * Once the plan is approved, it builds a kapp cache for all candidate kapps. 
    * Then `make install` is run against each kapp in the cache. 
      * By default kapps are installed in parallel, but single-threading can be specified per kapp in each manifest. This means that e.g. shared infrastructure can be installed first and can block the installation of all other kapps. Once the shared infra is up, the remaining kapps can be installed in parallel to reduce the amount of time necessary to provision clusters and apply the manifests.
      * Each kapp is also run in a planning mode and output is logged. This means that any kapps that use e.g. terraform can run `terraform plan`. 
      * After each kapp has been planned, all the log output can be merged and parsed. The CI pipeline can then halt the deployment and require manual approval if, e.g. any infrastructure is planned to be deleted.
      * Finally, each kapp is rerun in an `apply` mode to apply any previously generated plans. This can include applying any previously generated terraform plans if terraform is being used to manage infrastructure.
  * A command to build and maintain kapp caches.
  * A command to initialise kapps with dynamic configs (e.g. generate a terraform backend for the region the cluster will run in, etc.).
* Sample Jenkins pipelines - kapps should work without any dependency on, or awareness of the CI/CD system they're being deployed with. This makes kapps simpler to develop since it removes a dependency on Jenkins which is a pain to test locally.
  * CI/CD "business logic" should be kept to a minimum in Jenkins pipelines. It should do basic checks but beyond that should just call `sugarkube`. This makes is simple for developers to reproduce clusters and provision them outside their CI/CD system. 

We will also provide non-trivial sample kapps to demonstrate all of the above.

## Project Status
Sugarkube is at the inception phase. We shortly expect to have a proof-of-concept, then we'll start documenting the architecture and quickstarts. Once we have all of that we'll be ready for feedback and to roll out to early adopters for non-critical projects.

