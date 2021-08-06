---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "opentoolchain_tekton_pipeline_config Data Source - terraform-provider-opentoolchain"
subcategory: ""
description: |-
  Get tekton pipeline configuration
---

# opentoolchain_tekton_pipeline_config (Data Source)

Get tekton pipeline configuration

## Example Usage

```terraform
data "opentoolchain_tekton_pipeline_config" "tc" {
    env_id = "ibm:yp:us-east"
    guid   = var.pipeline_guid
}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- **env_id** (String) Environment ID, example: `ibm:yp:us-south`
- **guid** (String) The tekton pipeline `guid`

### Optional

- **id** (String) The ID of this resource.

### Read-Only

- **name** (String) Pipeline name
- **secret_env** (Map of String, Sensitive) Pipeline environment secret properties
- **text_env** (Map of String) Pipeline environment text properties
- **toolchain_crn** (String) The toolchain `crn`
- **toolchain_guid** (String) The toolchain `guid`
- **trigger** (Set of Object) (see [below for nested schema](#nestedatt--trigger))

<a id="nestedatt--trigger"></a>
### Nested Schema for `trigger`

Read-Only:

- **enabled** (Boolean)
- **event_listener** (String)
- **events** (List of Object) (see [below for nested schema](#nestedobjatt--trigger--events))
- **github_integration_guid** (String)
- **id** (String)
- **name** (String)
- **scm** (List of Object) (see [below for nested schema](#nestedobjatt--trigger--scm))
- **type** (String)

<a id="nestedobjatt--trigger--events"></a>
### Nested Schema for `trigger.events`

Read-Only:

- **pull_request** (Boolean)
- **pull_request_closed** (Boolean)
- **push** (Boolean)


<a id="nestedobjatt--trigger--scm"></a>
### Nested Schema for `trigger.scm`

Read-Only:

- **branch** (String)
- **hook_id** (String)
- **type** (String)
- **url** (String)

