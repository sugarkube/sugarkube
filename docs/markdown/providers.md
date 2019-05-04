# Providers
When you're configuring a [stack](stacks.md) you need to specify the 'provider'. Supported options are currently:

* aws - supports regions
* local - doesn't support regions

Providers (as in cloud providers) organise loading config files from disk. Some concepts (such as regions) don't make sense for local Minikube clusters so multiple ways of organising config files have been created to more closely mirror the target cloud a cluster will run on. 