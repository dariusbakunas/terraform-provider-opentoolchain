---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "opentoolchain_pipeline_triggers Data Source - terraform-provider-opentoolchain"
subcategory: ""
description: |-
  Get tekton pipline triggers (DEPRECATED)
---

# opentoolchain_pipeline_triggers (Data Source)

Get tekton pipline triggers (DEPRECATED)

## Example Usage

```terraform
data "opentoolchain_pipeline_triggers" "pt" {
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
- **pattern** (String)
- **type** (String)
- **url** (String)


