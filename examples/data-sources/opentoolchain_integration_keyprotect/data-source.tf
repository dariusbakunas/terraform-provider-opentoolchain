data "opentoolchain_integration_keyprotect" "kp" {
  toolchain_id = opentoolchain_toolchain.tc.guid
  integration_id = opentoolchain_integration_keyprotect.kp.integration_id
  env_id       = "ibm:yp:us-east"
}
