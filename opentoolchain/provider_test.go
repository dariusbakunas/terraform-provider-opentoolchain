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
var resourceGroupName string
var ibmRepoURL string
var kpInstanceName string
var envID string

const testResourcePrefix = "tf_acc_test"

func init() {
	resourceGroupID = os.Getenv("RESOURCE_GROUP_ID")
	resourceGroupName = os.Getenv("RESOURCE_GROUP_NAME")
	ibmRepoURL = os.Getenv("IBM_REPO_URL")
	kpInstanceName = os.Getenv("KP_INSTANCE_NAME")
	envID = os.Getenv("ENV_ID")

	if resourceGroupID == "" {
		resourceGroupID = "f6e4cda2a2844978aeeca5a44b584646"
		log.Printf("[INFO] Set the environment variable RESOURCE_GROUP_ID for testing Open Toolchain resources else it is set to default '%s'", resourceGroupID)
	}

	if resourceGroupName == "" {
		resourceGroupName = "test-resource-group"
		log.Printf("[INFO] Set the environment variable RESOURCE_GROUP_NAME for testing Open Toolchain resources else it is set to default '%s'", resourceGroupName)
	}

	if envID == "" {
		envID = "ibm:yp:us-east"
		log.Printf("[INFO] Set the environment variable ENV_ID for testing Open Toolchain resources else it is set to default '%s'", envID)
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

	if resourceGroupName == "" || resourceGroupID == "" {
		t.Fatal("RESOURCE_GROUP_NAME and RESOURCE_GROUP_ID must be set for acceptance tests")
	}
}
