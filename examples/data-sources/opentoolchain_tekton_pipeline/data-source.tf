data "opentoolchain_tekton_pipeline" "tp" {
  pipeline_id = opentoolchain_tekton_pipeline.tp.pipeline_id
  env_id       = "ibm:yp:us-east"
}
