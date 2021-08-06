data "opentoolchain_tekton_pipeline_config" "tc" {
    env_id = "ibm:yp:us-east"
    guid   = var.pipeline_guid
}
