# Outputs

Sometimes a kapp will create some shared infrastructure or resource that other kapps need. E.g. perhaps you want to run all multiple web sites using a single shared RDS database. The hostname of the database will only be assigned by AWS once it's created - it can't be known in advance.

In this scenario, you can use outputs to make value available between kapps. There are two steps to using kapps - declaring that a kapp produces an output, and then using it in other kapps.

## Declaring outputs

A kapp can declare that it creates outputs with the `outputs` block:

```yaml
# kapp: web:shared-database    (this is its fully-qualified ID)
outputs:
- id: demo-output
  format: yaml
  path: _generated_output.yaml
  sensitive: true
```

Multiple outputs can be declared.

Sugarkube will load the file at the declared `path` and try to parse it depending on the `format` - accepted values are `yaml`, `json` and `text`. If YAML or JSON are used, the output will be parsed and elements will be accessible using dot notation. If `text` is used, the file will be loaded  

If the output is marked as `sensitive`, the file will be deleted as soon as the output has been loaded. This is intended to keep secrets off disk as much as possible.

## Using outputs

Outputs are available in all templated files (e.g. manifest files, sugarkube.yaml files, etc.) under the `.outputs` key. Outputs are stored under multiple names for convenience:

* `this` - only accessible to the kapp that declared the output, and can be used in that kapps own templates (which are rendered both before running th ekapp and again after loading any outputs)
* `<kapp ID>` - only accessible by kapps in the same manifest
* `<manifest ID>:<kapp ID>` - the fully qualified kapp ID is used to access the output of kapps in different manifests.

Annoyingly because of how go treats hyphens and colons in templates you need to make some replacements. Make the following replacements to a kapp's fully-qualfied ID and/or output ID:

* Replace hyphens ('-') with a single underscore
* Replace colons (':') with two underscores

## Example

If the path `_generated_output.yaml` defined by `web:shared-database` above contained the following YAML:

```yaml
size: big
instances:
  large: 3
```

Various fields could be accessed as follows:

* By the kapp itself as `{{ .outputs.this.demo_output.size }}`
* By a kapp in the same (`web`) manifest as `{{ .outputs.shared_database.demo_output.instances.large }}`
* By a kapp in a different manifest as `{{ .outputs.web__shared_database.demo_output.instances.large }}`
