package opentoolchain

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccOpenToolchainToolchainDataSource_basic(t *testing.T) {
	envID := "ibm:yp:us-east"
	repository := "https://github.com/open-toolchain/empty-toolchain"
	name := fmt.Sprintf("%s_toolchain_%d", testResourcePrefix, acctest.RandIntRange(10, 100))

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: setupToolchainConfig(name, envID, resourceGroupID, repository),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.opentoolchain_toolchain.tc", "env_id", envID),
					resource.TestCheckResourceAttr("data.opentoolchain_toolchain.tc", "template_repository", repository),
					resource.TestCheckResourceAttr("data.opentoolchain_toolchain.tc", "name", name),
				),
			},
		},
	})
}

func setupToolchainConfig(name string, envID string, rgID string, repository string) string {
	return fmt.Sprintf(`
        resource "opentoolchain_toolchain" "tc" {
            env_id              = "%s"
            name                = "%s"
            resource_group_id   = "%s"
            template_repository = "%s"
            template_branch = "master"
        }
        data "opentoolchain_toolchain" "tc" {
            guid   = opentoolchain_toolchain.tc.guid
            env_id = opentoolchain_toolchain.tc.env_id
        }
    `, envID, name, rgID, repository)
}
