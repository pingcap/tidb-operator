terraform {
  required_version = ">= 0.12"
  required_providers {
    azurerm  = "~> 2.40"
    external = "~> 2.1"
    helm     = "~> 2.0"
    null     = "~> 3.1"
  }
}
