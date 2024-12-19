terraform {
  required_version = ">= 0.12"
  required_providers {
    aws      = "~> 2.27"
    external = "~> 1.2"
    helm     = "~> 0.10"
    null     = "~> 3.0"
    template = "~> 2.1"
  }
}
