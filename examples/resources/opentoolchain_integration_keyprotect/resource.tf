resource "opentoolchain_integration_keyprotect" "kp" {
  toolchain_id = opentoolchain_toolchain.tc.guid
  env_id       = "ibm:yp:us-east"
  resource_group = data.ibm_resource_group.rg.name
  instance_region = "ibm:yp:us-east"
  instance_name = "kp-instance-dev"
  name = "kp-integration"
}
