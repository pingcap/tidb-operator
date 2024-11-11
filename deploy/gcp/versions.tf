
terraform {
  required_version = ">= 0.12"
  required_providers {
    google      = "~> 2.16"
    google-beta = "~> 6.0"
    external    = "~> 1.2"
    helm        = "~> 0.10"
    null        = "~> 2.1"
  }
}
