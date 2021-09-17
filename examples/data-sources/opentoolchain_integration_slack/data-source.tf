data "opentoolchain_integration_slack" "sl" {
  toolchain_id = opentoolchain_toolchain.tc.guid
  integration_id = opentoolchain_integration_slack.sl.integration_id
  env_id       = "ibm:yp:us-east"
}
