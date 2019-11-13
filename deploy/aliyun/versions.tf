
terraform {
  required_version = ">= 0.12"
  required_providers {
    alicloud = "~> 1.56"
    external = "~> 1.2"
    helm     = "~> 0.10"
    null     = "~> 2.1"
    template = "~> 2.1"
  }
}
