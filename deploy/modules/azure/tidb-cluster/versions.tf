terraform {
  required_version = ">= 0.12"
  required_providers {
    azurerm  = "~> 2.50"
    external = "~> 2.1"
    local    = "~> 2.1"
    null     = "~> 3.1"
  }
}
