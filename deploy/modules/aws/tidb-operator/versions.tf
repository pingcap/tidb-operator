terraform {
  required_version = ">= 0.12"
  required_providers {
    aws      = "~> 2.27"
    helm     = "~> 2.0"
    local    = "~> 1.3"
    null     = "~> 2.1"
    template = "~> 2.1"
  }
}
