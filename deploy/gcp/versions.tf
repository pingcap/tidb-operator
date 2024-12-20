
terraform {
  required_version = ">= 0.12"
  required_providers {
    google      = "~> 2.16"
    google-beta = "~> 2.16"
    external    = "~> 1.2"
    helm        = "~> 2.0"
    null        = "~> 2.1"
  }
}
