data "opentoolchain_integration_ibm_github" "gt" {
  toolchain_id = opentoolchain_toolchain.tc.guid
  integration_id = opentoolchain_integration_ibm_github.gt.integration_id
  env_id       = "ibm:yp:us-east"
}
