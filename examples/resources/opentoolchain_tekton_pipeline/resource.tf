resource "opentoolchain_tekton_pipeline" "tp" {
  toolchain_id = opentoolchain_toolchain.tc.guid
  env_id       = "ibm:yp:us-east"
  name         = "main-pipeline"

  definition {
    github_integration_id = opentoolchain_integration_ibm_github.gi.integration_id
    github_url            = opentoolchain_integration_ibm_github.gi.repo_url
    branch                = "master"
    path                  = ".tekton"
  }

  text_env = {
    TEST_VAR = "example_value"
  }

  secret_env = {
    TEST_VAULT_SECRET = "{vault::${opentoolchain_integration_keyprotect.kp.name}.TEST_VAULT_SECRET}"
  }

  trigger {
    enabled = true
    name = "CI Manual Trigger"
    event_listener = "ci-manual"
    type = "manual"
  }

  trigger {
    enabled = false
    name = "CI Git PR Trigger"
    branch = "master"
    github_integration_id = opentoolchain_integration_ibm_github.gi.integration_id
    github_url = opentoolchain_integration_github.gi.repo_url
    event_listener = "ci-git-pr"
    on_pull_request = true
    type = "scm"
  }
}
