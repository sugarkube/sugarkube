# Sugarkube - Infrastructure Dependency Management

**TLDR;** Basically like `requirements.txt` or `package.json` but for 
infrastructure and applications. Not specific to Kubernetes but that's its
primary target.

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

Applications ("Kapps") just need to be versionable and have a Makefile with 
several standard targets to be compatible, which means if you can script it 
you can run it as a Kapp. 

Kapps should create all the infrastructure they need depending on where they're 
run. E.g. installing Chart Museum on a local Minikube cluster shouldn't create
an S3 bucket, but when it's run on AWS it should. Any infra used by more than
a single Kapp should be put into its own Kapp to simplify dependency management.

Sugarkube can also create Kubernetes clusters on various backends
(e.g. AWS, local, etc.) using a variety of provisioners (e.g. Kops, Minikube).

## Features
Use Sugarkube to:

  * Fully version your applications and infrastructure as "Kapps".
  * Automate creation and configuration of your infrastructure and Kapps from 
    scratch on multiple backends for full disaster recovery and reproducible/ephemeral environments.
  * Automate building differently specced ephemeral dev/test environments fully 
    configured with your core dependencies (e.g. Cert Manager, Vault, etc.) so 
    you can get straight to work.
  * Push your Kapps through a sane release pipeline. Develop locally or
    on (possibly ephemeral) dev clusters, test on staging, then release to one or 
    multiple target prod clusters. The process is up to you and Sugarkube is
    compatible with Jenkins.
  * Provide a multi-cloud and/or cloud exit strategy.
  * Split your infra/Kapps into layers. Create manifests for your core Kapps
    and for different dev teams to reflect how your organisation uses your 
    clusters. E.g. Dev Team A's dev/test clusters use 'Core' + 'KappA', but in 
    staging & prod you run 'Core' + 'KappA' + 'KappB' + 'Monitoring'.
  * Use community Kapps to immediately install e.g. a monitoring stack with
    Prometheus, Grafana, ElasticSearch, etc. then choose which alerting 
    Kapps to install on top. Because you can layer your manifests, this 
    monitoring stack only need be deployed in particular clusters so you don't 
    bloat local/dev clusters.

Sugarkube is great for new projects, but even legacy applications can be 
migrated into Kapps. You can migrate a bit at a time to see how it helps you.

## More info
See https://sugarkube.io for more info and documentation. 

## Status
Sugarkube is a work in progress and not ready for production use just yet.
