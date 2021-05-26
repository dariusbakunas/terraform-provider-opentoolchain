package opentoolchain

import (
	"log"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

var testAccProviders map[string]*schema.Provider
var testAccProvider *schema.Provider

var resourceGroupID string

const testResourcePrefix = "tf_acc_test"

func init() {
	resourceGroupID = os.Getenv("RESOURCE_GROUP_ID")
	if resourceGroupID == "" {
		resourceGroupID = "f6e4cda2a2844978aeeca5a44b584646"
		log.Printf("[INFO] Set the environment variable RESOURCE_GROUP_ID for testing Open Toolchain resources else it is set to default '%s'", resourceGroupID)
	}
}

func init() {
	testAccProvider = Provider()

	testAccProviders = map[string]*schema.Provider{
		"opentoolchain": testAccProvider,
	}
}

func testAccPreCheck(t *testing.T) {
	apiKey := os.Getenv("IAM_API_KEY")
	accessToken := os.Getenv("IAM_ACCESS_TOKEN")

	if apiKey == "" && accessToken == "" {
		t.Fatal("IAM_API_KEY or IAM_ACCESS_TOKEN env must be set for acceptance tests")
	}
}
