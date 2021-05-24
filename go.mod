module github.com/dariusbakunas/terraform-provider-opentoolchain

go 1.15

replace github.com/dariusbakunas/opentoolchain-go-sdk => /Users/darius/Projects/opentoolchain-go-sdk

require (
	github.com/IBM/go-sdk-core v1.1.0
	github.com/dariusbakunas/opentoolchain-go-sdk v0.0.0-20210524162749-064b79972c52
	github.com/hashicorp/terraform-plugin-sdk/v2 v2.6.1
	github.com/mattn/go-colorable v0.1.8 // indirect
)
