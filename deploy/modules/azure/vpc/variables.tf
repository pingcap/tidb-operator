variable "region" {
  description = "The region name"
}

variable "resource_group" {
  description = "The Azure resource group name"
}

variable "vpc_name" {
  description = "Name of the VPC"
}

variable "create_vpc" {
  description = "Create a new VPC or not"
  default     = true
}

# The Azure virtual network can be as large as /8, but is limited to 65,536 configured IP addresses.
variable "vpc_cidr" {
  description = "VPC cidr, must be set correctly if create_vpc is true"
  default     = ["10.0.0.0/16"]
}
