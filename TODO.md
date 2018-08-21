# To do
## Repo-related tasks
* Add licence
* Add copyright notice to the top of each file
* CLA (see https://github.com/heptio/ark)
* Code of conduct

## Code-related tasks
* Kops support
* Support acquiring manifests with the acquirers
* Create a filesystem acquirer. If there's no protocol, assume file:// 
* CLI flags to set the log level
* Print important info instead of logging it
* Structured logging - it works for tests but isn't being set up right for the 
  main binary.
* Add support for verifying signed tags
* More tests 
* See if we can suppress warning in overridden makefiles by using the technique
  by mpb [described here](https://stackoverflow.com/questions/11958626/make-file-warning-overriding-commands-for-target)

## Docs
* Create web site
* Fully functional examples
