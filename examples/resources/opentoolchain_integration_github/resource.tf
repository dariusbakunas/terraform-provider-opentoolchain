resource "opentoolchain_integration_github" "gt" {
  toolchain_id = opentoolchain_toolchain.tc.guid
  env_id       = opentoolchain_toolchain.tc.env_id
  enable_issues = true
  repo_url     = "https://github.com/<account>/repository"
}
