# Sugarkube - Infrastructure Dependency Management

**TL;DR** Basically like `requirements.txt` or `package.json` but for 
infrastructure and applications. Can be used to spin up and provision cloud
infrastructure from scratch and to deploy your applications onto it. Can be
used as a production release pipeline. Not specific to Kubernetes or Helm.

Check out the [sample project](https://github.com/sugarkube/sample-project) that 
launches a minikube cluster, installs nginx-ingress, cert-manager and 2 wordpress 
sites, and loads different sample data into each of them. All this with 3 commands.

## Overview
Sugarkube is dependency management for your infrastructure. 
While its focus is Kubernetes-based clusters, it can be used to deploy your
applications onto any scriptable backend.

Dependencies are declared in 'manifest' files which describe which version of
an application to install onto whichever backend, similar to a Python/pip
'requirements.txt' file,  NPM 'package.json' or Java 'pom.xml'. Therefore 
manifests can be versioned and are fully declarative. They describe which 
versions of which applications or infrastructure should be deployed onto 
whichever clusters/backends.

Applications ("kapps") just need to be versionable and have a Makefile with 
several standard targets to be compatible, which means if you can script it 
you can run it as a kapp. 

Kapps should create all the infrastructure they need depending on where they're 
run. E.g. installing Chart Museum on a local Minikube cluster shouldn't create
an S3 bucket, but when it's run on AWS it should. Any infra used by more than
a single kapp should be put into its own kapp to simplify dependency management.

Sugarkube can also create Kubernetes clusters on various backends
(e.g. AWS, local, etc.) using a variety of provisioners (e.g. Kops, Minikube).

## Features
Use Sugarkube to:

  * Fully version your applications and infrastructure as "kapps".
  * Automate creation and configuration of your infrastructure and kapps from 
    scratch on multiple backends to aid disaster recovery (excluding data) and 
    reproducible/ephemeral environments.
  * Automate building differently specced ephemeral dev/test environments fully 
    configured with your core dependencies (e.g. Cert Manager, Vault, etc.) so 
    you can get straight to work.
  * Push your kapps through a sane, idempotent release pipeline. Develop locally or
    on (possibly ephemeral) dev clusters, test on staging, then release to one or 
    multiple target prod clusters. The process is up to you and Sugarkube is
    compatible with Jenkins.
  * Provide a multi-cloud and/or cloud exit strategy.
  * Split your infra/kapps into layers. Create manifests for your core kapps
    and for different dev teams to reflect how your organisation uses your 
    clusters. E.g. Dev Team A's dev/test clusters use 'Core' + 'KappA', but in 
    staging & prod you run 'Core' + 'KappA' + 'KappB' + 'Monitoring'.
  * Use community kapps to immediately install e.g. a monitoring stack with
    Prometheus, Grafana, ElasticSearch, etc. then choose which alerting 
    kapps to install on top. Because you can layer your manifests, this 
    monitoring stack only need be deployed in particular clusters so you don't 
    bloat local/dev clusters.

Sugarkube is great for new projects, but even legacy applications can be 
migrated into kapps. You can migrate a bit at a time to see how it helps you.

## Status
Sugarkube is a work in progress and not ready for production use just yet.

# Installation & quick start
* Install [cfssl](https://github.com/cloudflare/cfssl) (on OSX run `brew install cfssl`)
* Download a [release](https://github.com/sugarkube/sugarkube/releases), or clone the repo and run `make build`.
* Clone the [sample project](https://github.com/sugarkube/sample-project).
* Launch a local minikube cluster (it may take a little while to come online):
```
  ./sugarkube cluster create -s sample-project/stacks.yaml -n local-web \
    -v --log-level=info
```

* Download the kapps to be installed into a local cache directory (`caches/local-web` 
in the below command). Have a poke around in this later to see how it works:
```
  ./sugarkube cache create -s sample-project/stacks.yaml -n local-web \
    caches/local-web -v --log-level=info 
```

* Install the kapps into the cluster:
```
  ./sugarkube kapps install -s sample-project/stacks.yaml -n local-web \
    ./caches/local-web --one-shot 
```

You should now have a local minikube cluster set up with Tiller installed
(configured for RBAC), along with Cert Manager, Nginx ingress, and two 
Wordpress sites, both with customised content in them. Access them with:

```
 minikube service -n wordpress-site1 --https wordpress-site1-wordpress
```
and
```
 minikube service -n wordpress-site2 --https wordpress-site2-wordpress
```
 
If you add the hostnames to your `/etc/hosts` file you should be able to access 
them through that (WIP).
```
echo $(minikube ip) wordpress.localhost | sudo tee -a /etc/hosts
```
~~And visit `https://wordpress.localhost`.~~

## Explore
Have a look in `sample-project/stacks.yaml` and look for `local-web`. Follow the
paths it declares. Then look in `./caches/local-web` to see how that relates to the
git repos defined in the manifests used in `stacks.yaml`.

# FAQ
### Does this depend on containers?
No. Sugarkube just acquires and runs versioned Makefiles. It's up to you 
what they do. 

### How does this compare to Helm?
Helm installs individual applications (e.g. Wordpress) but doesn't let you 
install a suite of related applications (e.g. a monitoring stack with 
Prometheus, Grafana, Elastic Search and various beats like filebeat, 
metricbeat, etc.). You could create a 'superchart' that defines all these as
dependencies, but then you have no control over just installing a subset of 
your charts.

Sugarkube gives you a simple way to specify, for example, that ChartA at 
version a.b.c should be installed alongside ChartB at version x.y.z.

Helm charts also don't create any required infrastructure. An application
like Chart Museum should probably be backed by S3 in production (if you're on
AWS), but Helm leaves it up to you to work out how to create that S3 bucket.
This gets more complicated if you have further dependencies like needing to 
first create KMS keys for S3 bucket encryption.

Because Sugarkube was written with provisioning from bare metal in mind you 
can configure it to install kapps that create shared infrastructure resources 
like load balancers, CloudFront distributions and KMS keys. These can then be 
used by e.g. Chart Museum, which could use the KMS key to create it's own S3 
bucket using Terraform/awscli.  

### How does this compare to Terraform?
Terraform is great for creating infrastructure, but using it to install Helm
charts should be an anti-pattern: 

  * Local development becomes a pain because you need to continually change
    the CLI args to make it reinstall a chart since it has no other way of 
    knowing whether a chart has changed and needs reinstalling.
  * You lose the ability to run `helm lint` because templating needs to be 
    done through Terraform.
  * You often end up with lots of related repositories that are a pain to
    version together if you want to split out your values from your terraform
    code/modules.

Sugarkube's sample kapps use Terraform for what it's good for - managing 
infrastructure, e.g. creating buckets, Route53 entries, load balancers, etc.

This means that it becomes possible to easily version your Terraform configs 
alongside the chart that needs them. E.g. if your Wordpress chart changes from 
being backed by Postgres on an EC2 to using RDS, the point at which that change
happens is tightly bound to actually creating the RDS instance. Since Sugarkube
kapps create the infrastructure they need, and manifests allow multiple kapps
to be versioned alongside each other, you could either update your Wordpress
kapp to create an RDS instance just for its own use, or create it in a shared 
kapp for multiple Wordpress instances to use.

Another benefit is that since Sugarkube delegates to `make`, Makefiles can be 
written that don't create any infrastructure at all when running on local 
minikube clusters, or that create smaller instances of, e.g. RDS databases while
developing.

### What if I don't use Kubernetes?
Sugarkube is really several things:

  * A set of conventions for creating applications with standard `make` targets 
    ("kapps").
  * A system for checking out related versions of related kapps from manifest files.
  * A process for checking which kapps need installing/deleting based on the 
    input manifest(s) and what's running on the target cluster, and running 
    `make install` or `make delete` on each one. It generates a few files first 
    and passes some extra parameters so the kapp has all the info it needs to 
    install/delete itself the right way. 

There's no hard dependency on Kubernetes. If you can install something with 
`make`, you should be able to convert it to a kapp to be installed by Sugarkube.

Sugarkube does treat Helm & Terraform code as first-class citizens so that, e.g. 
environment-specific `values-<env>.yaml` files can be passed as parameters, but 
these are extra benefits, not requirements to use Helm or Terraform.

As a convenience it can also launch clusters parameterised hierarchically. I.e.
values for launching clusters can be defined and overridden at various levels
to allow all dev clusters to have similar defaults different from your staging 
and prod clusters, but for everything to be overrideable.

### Where can I find more info?
See [https://www.sugarkube.io](https://www.sugarkube.io) for more info and 
documentation (in progress). 
