data "opentoolchain_toolchain" "tc" {
  guid   = "c282ae79-29e5-49f6-b885-9d2e58fd9e61"
  env_id = "ibm:yp:us-east"
}

locals {
  pipeline_name = data.opentoolchain_toolchain.tc.name
  pipeline_id   = coalesce([for svc in data.opentoolchain_toolchain.tc.services : svc.instance_id if svc.service_id == "pipeline" && lookup(svc.parameters, "type", "") == "tekton" && lookup(svc.parameters, "name", "") == local.pipeline_name]...)
}

resource "opentoolchain_pipeline_properties" "tp" {
  env_id = "ibm:yp:us-east"
  guid   = local.pipeline_id
  text_env = {
      INPUT_GIT_BRANCH: "test"
  }
}
