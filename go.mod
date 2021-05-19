module github.ibm.com/dbakuna/terraform-provider-opentoolchain

go 1.15

replace github.ibm.com/dbakuna/opentoolchain-go-sdk => /Users/darius/Projects/opentoolchain-go-sdk

require (
	github.com/IBM/go-sdk-core v1.1.0
	github.com/hashicorp/terraform-plugin-sdk/v2 v2.6.1
	github.ibm.com/dbakuna/opentoolchain-go-sdk v0.0.0-20210519181904-a81a6c82a317
)
