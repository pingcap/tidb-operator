variable "region" {
  description = "Azure region"
}

variable "resource_group" {
  description = "The resource group of this key"
  type        = string
}

variable "name" {
  description = "Unique name for the key, should also be a valid filename. This will prefix the public/private key."
}

variable "path" {
  description = "Path to a directory where the public and private key will be stored."
  default     = ""
}
