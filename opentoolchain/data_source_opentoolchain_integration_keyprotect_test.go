package opentoolchain

import (
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"testing"
)

func TestAccOpenToolchainIntegrationKeyProtectDataSource_basic(t *testing.T) {
	toolchainName := fmt.Sprintf("%s_app_%d", testResourcePrefix, acctest.RandIntRange(10, 100))

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: setupOpenToolchainIntegrationKeyProtectDataSourceConfig(envID, toolchainName, resourceGroupID, kpInstanceName, resourceGroupName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.opentoolchain_integration_keyprotect.kp", "instance_region", envID),
					resource.TestCheckResourceAttr("data.opentoolchain_integration_keyprotect.kp", "instance_name", kpInstanceName),
					resource.TestCheckResourceAttr("data.opentoolchain_integration_keyprotect.kp", "name", "test-kp-integration"),
					resource.TestCheckResourceAttr("data.opentoolchain_integration_keyprotect.kp", "resource_group", resourceGroupName),
				),
			},
		},
	})
}

func setupOpenToolchainIntegrationKeyProtectDataSourceConfig(envID, toolchainName, resourceGroupID, kpInstanceName, rgName string) string {
	return fmt.Sprintf(`
		resource "opentoolchain_toolchain" "tc" {
			env_id            = "%s"
			name              = "%s"
			resource_group_id = "%s"
		}

		resource "opentoolchain_integration_keyprotect" "kp" {
			toolchain_id = opentoolchain_toolchain.tc.guid
			env_id       = opentoolchain_toolchain.tc.env_id
			instance_region = opentoolchain_toolchain.tc.env_id
			instance_name = "%s"
			resource_group = "%s"
			name = "test-kp-integration"
		}

		data "opentoolchain_integration_keyprotect" "kp" {
			toolchain_id = opentoolchain_toolchain.tc.guid
			integration_id = opentoolchain_integration_keyprotect.kp.integration_id
			env_id       = opentoolchain_toolchain.tc.env_id
		}
	`, envID, toolchainName, resourceGroupID, kpInstanceName, rgName)
}
