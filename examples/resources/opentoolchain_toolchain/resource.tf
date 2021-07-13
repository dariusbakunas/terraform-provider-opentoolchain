resource "opentoolchain_toolchain" "tc" {
  env_id              = "ibm:yp:us-east"
  resource_group_id   = "<resource group id>"
  repository_token    = var.git_api_key
  template_repository = "https://github.ibm.com/whc-toolchain/whc-developer-toolchain-CI"
  template_branch = "stable-3.3.3"
  tags = ["dev", "tf"]
  template_properties = {
    "toolchain.name" = "TEST"
    "apiKey" = var.ic_api_key
    "prodRegion" = "us-south"
    "clusterName" = "mycluster-free"
    "clusterNamespace" = "dev"
    "dockerConfigJson" = var.docker_config_json
    "inputGitBranch" = "master"
    "gitrepourl" = "https://github.ibm.com/<your repository>"
    "form.pipeline.parameters.UMBRELLA_GIT_BRANCH" = "dev"
    // add more properties here, depending on the template used
  }
}
