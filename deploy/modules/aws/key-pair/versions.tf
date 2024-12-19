terraform {
  required_version = ">= 0.12"
  required_providers {
    aws   = "~> 2.27"
    local = "~> 1.3"
    null  = "~> 3.0"
    tls   = "~> 2.1"
  }
}
