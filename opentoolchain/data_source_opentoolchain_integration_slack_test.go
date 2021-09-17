package opentoolchain

import (
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"testing"
)

func TestAccOpenToolchainIntegrationSlackDataSource_basic(t *testing.T) {
	toolchainName := fmt.Sprintf("%s_slack_%d", testResourcePrefix, acctest.RandIntRange(10, 100))

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: setupOpenToolchainIntegrationSlackDataSourceConfig(envID, toolchainName, resourceGroupID, slackWebhookURL, slackChannelName, slackTeamName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.opentoolchain_integration_slack.sl", "channel_name", slackChannelName),
					resource.TestCheckResourceAttr("data.opentoolchain_integration_slack.sl", "team_name", slackTeamName),
					resource.TestCheckResourceAttr("data.opentoolchain_integration_slack.sl", "events.#", "1"),
					resource.TestCheckResourceAttr("data.opentoolchain_integration_slack.sl", "events.0.pipeline_start", "false"),
					resource.TestCheckResourceAttr("data.opentoolchain_integration_slack.sl", "events.0.pipeline_success", "true"),
					resource.TestCheckResourceAttr("data.opentoolchain_integration_slack.sl", "events.0.pipeline_fail", "true"),
					resource.TestCheckResourceAttr("data.opentoolchain_integration_slack.sl", "events.0.toolchain_bind", "false"),
					resource.TestCheckResourceAttr("data.opentoolchain_integration_slack.sl", "events.0.toolchain_unbind", "true"),
				),
			},
		},
	})
}

func setupOpenToolchainIntegrationSlackDataSourceConfig(envID, toolchainName, resourceGroupID, webhookURL, channelName, teamName string) string {
	return fmt.Sprintf(`
		resource "opentoolchain_toolchain" "tc" {
			env_id            = "%s"
			name              = "%s"
			resource_group_id = "%s"
		}

		resource "opentoolchain_integration_slack" "sl" {
			toolchain_id = opentoolchain_toolchain.tc.guid
			env_id       = opentoolchain_toolchain.tc.env_id
			webhook_url = "%s"
			channel_name = "%s"
			team_name = "%s"

			events {
				pipeline_start = false
				toolchain_bind = false
			}
		}

		data "opentoolchain_integration_slack" "sl" {
			toolchain_id = opentoolchain_toolchain.tc.guid
			integration_id = opentoolchain_integration_slack.sl.integration_id
			env_id       = opentoolchain_toolchain.tc.env_id
		}
	`, envID, toolchainName, resourceGroupID, webhookURL, channelName, teamName)
}
