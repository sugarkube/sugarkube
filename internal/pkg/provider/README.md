# (Infrastructure/Cloud) Providers
This package represents different providers of infrastructure. While that 
obviously includes cloud providers llke AWS and Google, it could also represent
custom infrastructure hosted in a particular data centre with e.g. Rackspace, 
etc. 

So while a Provider is the entity you pay to host your infrastructure, 
`provisioners` control setting up infrastructure. Kapps are then installed
using the configured `installer` depending on your setup.

This package provides semantics per cloud provider. E.g. whereas AWS divides 
their infrastructure by account, region and availability zone, those terms 
don't make any sense for local (e.g. minikube) clusters or custom, non-HA 
data centres with a single location, etc. These semantics are used to target
clusters, parse CLI flags and to pass relevant environment variables to kapps
running on different providers.
