
terraform {
  required_version = ">= 0.12"
  required_providers {
    external = "~> 2.0"
    helm     = "~> 0.10"
    null     = "~> 2.1"
  }
}
