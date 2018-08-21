File names have no bearing on the order of execution. They are just prefixed with
numbers to make it easier to reason about which order manifests will be applied 
in, but the actual installation order is determined by the stack configuration 
(see `../stacks.yaml`, or by CLI args).

Different manifests are will be installed into different stacks, and will be 
parameterised differently depending on the actual environment.

Manifests are processed sequentially, but the contents of each manifest is processed
in parallel. Therefore an easy way to control parallelisation is to just create
another manifest file if you need to install a kapp single-threaded.
