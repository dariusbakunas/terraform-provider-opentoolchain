data "opentoolchain_integration_pagerduty" "pd" {
  toolchain_id = opentoolchain_toolchain.tc.guid
  env_id       = "ibm:yp:us-east"
  integration_id = opentoolchain_integration_pagerduty.pd.integration_id
}
