resource "opentoolchain_pipeline_triggers" "pt" {
    env_id = "ibm:yp:us-east"
    guid   = var.pipeline_guid

    trigger {
        name = "Manual Trigger Test"
        enabled = false
    }

    trigger {
        name = "Git Trigger"
        enabled = true
    }
}
