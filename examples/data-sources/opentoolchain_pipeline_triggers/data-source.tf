data "opentoolchain_pipeline_triggers" "pt" {
    env_id = "ibm:yp:us-east"
    guid   = var.pipeline_guid
}
