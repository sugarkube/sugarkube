# Pre-launch infra
* Create KMS keys in a kapp with an alias defined by a naming convention
* Allow the naming convention to be specified in sugarkube's config
* Use it to generate the KMS key's alias and to retrieve it using 
  a terraform datablock
* Use the extracted ID to populate terraform backend files.
* Perhaps we should cache the KMS key's ARN so we don't have to continually
  fetch it.
  
This pattern should be applicable to most infra. Even if we need to create some
in a pre-launch kapp, other kapps (and sugarkube) should be able to retrieve 
ARNs etc, by looking the resources up by name.
