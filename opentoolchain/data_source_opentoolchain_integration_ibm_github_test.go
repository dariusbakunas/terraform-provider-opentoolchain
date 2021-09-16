package opentoolchain

import (
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"testing"
)

func TestAccOpenToolchainIntegrationIBMGithubDataSource_basic(t *testing.T) {
	toolchainName := fmt.Sprintf("%s_app_%d", testResourcePrefix, acctest.RandIntRange(10, 100))

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: setupOpenToolchainIntegrationGithubDataSourceConfig(envID, toolchainName, resourceGroupID, ibmRepoURL),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.opentoolchain_integration_ibm_github.gt", "repo_url", fmt.Sprintf("%s.git", ibmRepoURL)),
					resource.TestCheckResourceAttr("data.opentoolchain_integration_ibm_github.gt", "private", "false"),
					resource.TestCheckResourceAttr("data.opentoolchain_integration_ibm_github.gt", "enable_issues", "true"),
				),
			},
		},
	})
}

func setupOpenToolchainIntegrationGithubDataSourceConfig(envID string, toolchainName string, resourceGroupID string, repoURL string) string {
	return fmt.Sprintf(`
		resource "opentoolchain_toolchain" "tc" {
			env_id            = "%s"
			name              = "%s"
			resource_group_id = "%s"
		}

		resource "opentoolchain_integration_ibm_github" "gt" {
			toolchain_id = opentoolchain_toolchain.tc.guid
			env_id       = opentoolchain_toolchain.tc.env_id
			enable_issues = true
			repo_url     = "%s"
		}

		data "opentoolchain_integration_ibm_github" "gt" {
			toolchain_id = opentoolchain_toolchain.tc.guid
			integration_id = opentoolchain_integration_ibm_github.gt.integration_id
			env_id       = opentoolchain_toolchain.tc.env_id
		}
	`, envID, toolchainName, resourceGroupID, repoURL)
}
