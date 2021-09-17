resource "opentoolchain_integration_slack" "sl" {
  toolchain_id = opentoolchain_toolchain.tc.guid
  env_id       = "ibm:yp:us-east"
  webhook_url  = "https://hooks.slack.com/services/XXXXXXXX"
  channel_name = "notifications"
  team_name = "yourslackteam"

  events {
    toolchain_bind = false
    toolchain_unbind = false
  }
}
