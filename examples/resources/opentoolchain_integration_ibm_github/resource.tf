resource "opentoolchain_integration_ibm_github" "gt" {
  toolchain_id = opentoolchain_toolchain.tc.guid
  env_id       = opentoolchain_toolchain.tc.env_id
  enable_issues = true
  repo_url     = "https://github.ibm.com/whc-developer-CI"
}
