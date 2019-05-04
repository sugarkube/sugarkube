# Introduction to Sugarkube

Welcome to this guide. This is intended for people who want to understand what problems Sugarkube solves and why it exists. A brief TLDR summary is: 

* Sugarkube is a release system that ships your applications and infrastructure code at the same time
* It doesn't require Kubernetes but provides extra features if you use it
* It works with any infrastructure you can script (cloud, local, on-prem, legacy) 
* It's compatible with any programming language
* It enables a multi-cloud strategy
* Sugarkube itself is a single golang binary that can be used for local development and be embedded in CI/CD pipelines. This simplifies local development by allowing developers to run exactly what the CI/CD tool will run.
* You can adopt it bit-by-bit while you test it out - you don't need to dedicate 3 months to migrate for an all-or-nothing release
* The project is currently in alpha
* Examples in these guides use Terraform and Helm, but both can be replaced by other similar tools as necessary (but things might be a bit tricky while we're still in alpha)

# The problem
Releasing software is difficult, especially when applications require certain infrastructure or cloud services, and their exact dependencies evolve over time. Coordinating infrastructure changes with code changes manually can work in the short term but be difficult as projects get more complex. Developers are typically under pressure to deliver products or prototypes rather than having several months to create a robust release pipeline. 

The outcome is that early decisions and processes that work with a very small team don't scale as the size of the team and complexity of the project increase. This can lead to a point in a project where the delivery rate drops to a crawl and developers end up fighting the release system, manually creating environments instead of just getting on with creating new features.

## Releases aren't that unique
We believe that there's a lot of commonality across release processes at different organisations. You may be tempted to think your stack is particularly unique, but unless you're operating at massive scale your release process can probably be summarised as:

> Release versions x, y and z of applications a, b and c through environments e, f and g, after possibly creating various bits of infrastructure in each environment first.

If that sounds like your organisation you're in luck. That's what Sugarkube aims to help you with!

# The ideal solution
We believe an ideal release pipeline should:

 * make it possible to spin up and tear down environments quickly and easily. This facilitates testing and makes your entire infrastructure more robust by preventing snowflakes (brittle environments with custom changes)
 * be able to easily reproduce the state of any cluster at an arbitrary point in time into a different cloud account
 * give you confidence in what you're releasing to prod by having tested that exact code in a lower environment (e.g. staging) 
 * scale to allow individual developers to work in isolated environments - either on dedicated clusters or in isolated parts of larger development clusters.
 * let developers get to work quickly on developing new features or fixing bugs instead of having to waste large amounts of time setting up dev/test clusters and cloud infrastructure first
 * allow developers to work locally as much as possible before developing in the Cloud 
 * not require you to use a particular CI/CD system during local development or testing in non-live environments (e.g. if your release pipeline is a custom Jenkins library, being forced to always deploy through Jenkins can complicate and slow down development)
 
 This can largely be summarised as:
 
 > Developers should be able to concentrate on coding, not fighting the release process.

# What is Sugarkube?
Sugarkube is a software release system that bundles your application code along with code to create any infrastructure it depends on, and versions it as a single unit. This means the releasable artefact is your application code + code to create dependent infrastructure. Because "app" is an overloaded term in software development, we call these bundles of applications and infrastructure code "kapps" (originally from "Kubernetes app", but there's no requirement to use Sugarkube with Kubernetes any more).

Many other tools either only create infrastructure (Kubernetes, Terraform, CloudFormation, etc.) or release your applications (Helm, tarballs, whatever). This means you need some way of coordinating your application changes with the infrastructure they depend on, which can be complicated and error-prone.

Sugarkube is primarily an orchestrator. After defining which of your applications (kapps) depend on which others Sugarkube constructs a dependency graph which allows it to install your kapps in the correct order and uninstall them in the correct order. E.g. if your wordpress site depends on a database and load balancer, Sugarkube can make sure they're created first and deleted after wordpress has been deleted when uninstalling.

Versioning your applications and infrastructure together is incredibly powerful. This idea means Sugarkube can:

* Recreate your clusters at any point in time (minus data of course). Therefore Sugarkube allows you to easily create short-lived ephemeral clusters on demand. This ability alone - which is entirely optional - opens up some very powerful capabilities such as:
    * Allowing each developer or team to have their own dev clusters, and tear them down when they're done.
    * Spin up/tear down testing/staging clusters.
    * Frequently test your disaster recovery processes, since if you're regularly creating and tearing down your clusters you'll reduce the risk of uncommitted adhoc/manual changes tainting your cluster.
    * Simplify going multi-region - if you're already deploying your cluster to one cloud region it's simple to deploy to another if you've followed the best practices
    * Ease cluster upgrades - you could bring up a new instance of your prod cluster in a non-live account to test a Kubernetes rolling upgrade before applying it in prod, or go one step further and use a blue/green release process. In that case you could create an entirely new prod cluster in your prod account and gradually shift more and more traffic onto it. If all goes well you could tear down your old cluster, or if not just redirect all traffic back to the original.
    * Aid compliance with e.g. PCI requirements by making your deliverable artefact your entire cluster. When dealing with PCI regulations the less that's running in your cluster and the less connectivity it has, the better.
* Support multi-cloud - A kapp can create one set of infrastructure when being installed into AWS and a different set when being installed into GCP/Azure or even on-premise or locally. Just write the appropriate scripts/Terraform configs and Sugarkube will run the correct ones depending on your target cloud.
* Manage exactly which versions of your kapps (bundles) get released into each of your environments.
* Truly promote your applications and infrastructure through environments. An emphasise on portable artefacts (kapps) prevents you creating brittle snowflake environments.
* Install "slices" of your stack into different environments. For example if you have several monitoring and metric collection applications installed alongside your web sites, you can choose not to install the monitoring stack in your dev environment if you're not going to work on it.

All of the above make it simple to start working on new features without wasting time recreating infrastructure that your applications need. Adopting Sugarkube for a project will also lay extensive foundations for future scaling, both technologically and for when your team grows. 

### Hierarchical configuration
A key features that makes Sugarkube so flexible is that it allows you to define your configuration hierarchically. This means you can create default configurations and progressively override it or replace certain parts at more specific levels. To give a concrete example, a directory structure like such as below allows you to override and tailor your configurations in several ways:

```
- providers
  - aws
    - dev
      - kops.yaml   <- extra config just for dev clusters
      - dev1.yaml   <- configs just for the 'dev1' cluster
      - eu-west-1.yaml   <- configs for all clusters running in eu-west-1
    - test
      - kops.yaml   <- extra config just for test clusters
    kops.yaml    <- default kops config
```

Sugarkube will search for files with various basenames at each level in your directory hierarchy, merging them together depending on the parameters of your target cluster. In addition these files are also golang templates. Some practical applications of this are:

* Easily configure using a few, small instances for your cluster in dev/test environments while using more, larger instances in staging and production.
* Easily set default, region-specific base image IDs.
* Apply naming conventions to fully namespace all your resources - e.g. create DNS zones per cluster called `{{cluster}}.{{region}}.example.com` 

### Extra features for Kubernetes users
If you work with Kubernetes clusters Sugarkube provides additional features. As mentioned above, it can launch Kubernetes clusters and can do so with several provisioners, for now Kops and Minikube, and then configure them. For example it can patch Kops YAML configs before triggering an update to apply those changes. This makes it a useful tool for administering Kubernetes clusters. However its main benefit is that it allows you to create a cluster and install your applications (with dependent infrastructure) with a single command.

### Choose your own tools
As an orchestrator Sugarkube wraps other tools you're probably already familiar with - Kops/Minikube and Make. We provide a standard set of Makefiles for working with Helm/Terraform code, but you aren't forced to use any particular set of tools or technologies. Sugarkube will work with on-premise, legacy systems and infrastructure provided it's scriptable, and also with any programming language. You can adopt it bit-by-bit while you get used to it, and migrate more to it (or drop it) as you wish.

### Dealing with shared infrastructure 
One important thing to point out is that kapps must only create infrastructure that is only used by the application in the kapp. Any infrastructure that's shared between multiple applications/kapps must be created by another kapp (i.e. so you have one or several kapps dedicated to creating shared infrastructure like load balancers, hosted zone records, etc.).
These 'shared infrastructure' kapps therefore form the foundation for running certain groups of applications. So for example, you could create a shared infrastructure kapp to create your load balancer and hosted zone records, and configure it to be executed before executing kapps to install your web applications, etc.

# Kapps
The bundles of versioned application + infrastructure code are called "kapps". They're simply git repos where different directories contain a Makefile with some predefined targets, and a `sugarkube.yaml` config file that configures the kapp. The git repos for kapps are tagged to create different versions.

If you decide to install your applications into a Kubernetes cluster as a Helm chart and manage your infrastructure using Terraform code you can take advantage of our ready-made Makefiles that should cover 80-90% of use-cases. However, you have complete freedom to implement Makefiles as you want with several minor caveats. When Sugarkube runs it'll pass several environment variables to the Makefile to allow it to modify its behaviour depending on which cloud provider is being targetted, the name of the target cluster, etc. 

**Note**: Although you don't have to use Kubernetes, Helm and Terraform with Sugarkube, we've made an assumption that you will while we're still in alpha. This allows us to simplify the problem-space and get something working in a more predictable setup instead of trying to please everyone immediately. So if you choose not to use K8s, Helm and Terraform you may find a few things don't work as expected. Please open an issue on Github to tell us about those scenarios if you run into them so we can track them. 

## Alpha software
To reiterate, Sugarkube is currently in alpha. To speed up development and to simplify the problem-space, examples for Sugarkube use Helm and Terraform. Helm is used because Kubernetes is becoming more and more popular so it makes sense to target it. Terraform is used because it will delete extraneous infrastructure, making it easy to tear down infrastructure. This is useful for testing and resetting environments. Despite this, it should be possible to use other tools as necessary but it might not be plain sailing for now. In future we hope to remove any dependencies on Helm, Terraform and Make.
