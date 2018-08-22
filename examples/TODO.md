# Examples
## Stacks
Create some sample stacks:
* Large/small minikube
* Large/small kops
  * Make it simple to select different networking configs
  * Demonstrate overriding Kops configuration
  * Demo updating Kops
* Kops with public/private API servers
  * Expose the private one with a public shared LB

## Manifests
Come up with a few different manifests for different layers, e.g.:

* K8s bootstrap ✔
  * Tiller ✔

* Core ✔
  * Nginx-ingress ✔
  * Cert manager ✔

* AWS ✔
  * Kiam ✔ (untested)

* Monitoring
  * Prometheus
  * Grafana

* Backups
  * Ark
    * Demonstrate using Kiam and also hard-coded credentials and/or Vault 

* Security
  * Keycloak
    * Uses H2 DB locally
    * Uses RDS on AWS specced differently per environment (e.g. dev/prod, etc.)
  * Vault
  * Kuberos

* CI/CD:
  * Jenkins
  
* Web:
  * Wordpress (multiple instances sharing the same DB, plus multiple with 
    their own DBs)
    * Backed by RDS and/or MariaDB
  * Memcached 
    * locally  ✔
    * elasticache on AWS
