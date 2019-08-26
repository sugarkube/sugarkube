# Sugarkube - Infrastructure Dependency Management

**TL;DR** Basically like `requirements.txt` or `package.json` but for 
infrastructure and applications. Can be used to spin up and provision cloud
infrastructure from scratch and to deploy your applications onto it. Can be
used as a production release pipeline. Not specific to Kubernetes or Helm.

Check out the [sample project](https://github.com/sugarkube/sample-project) that 
launches a minikube cluster, installs nginx-ingress, cert-manager and 2 wordpress 
sites, and loads different sample data into each of them. All this with a few commands.

Read the [complete documentation](https://docs.sugarkube.io).

**Create a cluster**
![Create a cluster](docs/svgs/cluster-create.svg)

**Create a workspace**
![Create a cluster](docs/svgs/workspace-create.svg)

**Install stuff**
![Create a cluster](docs/svgs/kapps-install.svg)

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

Applications ("kapps") just need to be versionable and have a `sugarkube.yaml` file 
that configure it. If you can script something it you can run it as a kapp. 

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
Sugarkube is a work in progress but ready for early adopters who don't mind things being a bit clunky.

# Installation & quick start
* Install [cfssl](https://github.com/cloudflare/cfssl) (on OSX run `brew install cfssl`)
* Download a [release](https://github.com/sugarkube/sugarkube/releases), or clone the repo and run `make build`.
* Clone the [sample project](https://github.com/sugarkube/sample-project).
* Launch a local minikube cluster (it may take a little while to come online):
```
  ./sugarkube cluster create -s sample-project/stacks.yaml -n local-web \
    -v --log-level=info
```

* Download the kapps to be installed into a local workspace directory (`workspaces/local-web` 
in the below command). Have a poke around in this later to see how it works:
```
  ./sugarkube workspace create -s sample-project/stacks.yaml -n local-web \
    workspaces/local-web -v --log-level=info 
```

* Install the kapps into the cluster:
```
  ./sugarkube kapps install -s sample-project/stacks.yaml -n local-web \
    ./workspaces/local-web --one-shot 
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
paths it declares. Then look in `./workspaces/local-web` to see how that relates to the
git repos defined in the manifests used in `stacks.yaml`.

### Where can I find more info?
See [https://www.sugarkube.io](https://www.sugarkube.io) for more info and 
complete documentation. 
