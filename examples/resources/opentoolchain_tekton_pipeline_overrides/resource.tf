resource "opentoolchain_tekton_pipeline_overrides" "ov" {
    env_id = "ibm:yp:us-east"
    guid   = var.pipeline_guid

    text_env = {
        INPUT_GIT_BRANCH: "modified",
    }

    secret_env = {
        VAULT_SECRET: "{vault::vault_integration_name.VAULT_KEY}"
    }

    trigger {
        name = "Manual Trigger"
        enabled = true
    }

    trigger {
        name = "Git Trigger"
        enabled = false
    }
}
