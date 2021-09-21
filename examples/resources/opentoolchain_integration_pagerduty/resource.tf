resource "opentoolchain_integration_pagerduty" "pd" {
  toolchain_id = opentoolchain_toolchain.tc.guid
  env_id       = "ibm:yp:us-east"
  api_key      = var.api_key
  service_name = "TF Test"
  primary_email = "test@gmail.com"
  primary_phone_number = "1112223333"
}
