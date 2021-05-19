terraform {
  required_providers {
    opentoolchain = {
      source = "ibm.com/dbakuna/opentoolchain"
      version = "0.0.1"
    } 
  }
}

variable "iam_access_token" {
    type = string
}

provider "opentoolchain" {  
    iam_access_token = var.iam_access_token    
}

data "opentoolchain_toolchain" "tc" {
    guid = "60abcfc4-d6bc-47a5-99c2-039fc03f9ab2"
    env_id = "ibm:yp:us-east"
}

output "data" {
    value = data.opentoolchain_toolchain.tc
}