
terraform {
  required_version = ">= 0.12"
  required_providers {
    external = "~> 1.2"
    helm     = "~> 0.10"
    null     = "~> 2.1"
  }
}
