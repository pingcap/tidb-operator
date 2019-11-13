variable "region" {
  description = "AWS region"
}

variable "vpc_name" {
  description = "Name of the VPC"
}

variable "create_vpc" {
  description = "Create a new VPC or not, if true the vpc_id/subnet_ids must be set correctly, otherwise the vpc_cidr/private_subnets/public_subnets must be set correctly"
  default     = true
}

variable "vpc_cidr" {
  description = "VPC cidr, must be set correctly if create_vpc is true"
  default     = "10.0.0.0/16"
}

variable "private_subnets" {
  description = "VPC private subnets, must be set correctly if create_vpc is true"
  type        = list(string)
  default     = ["10.0.16.0/20", "10.0.32.0/20", "10.0.48.0/20"]
}

variable "public_subnets" {
  description = "VPC public subnets, must be set correctly if create_vpc is true"
  type        = list(string)
  default     = ["10.0.64.0/20", "10.0.80.0/20", "10.0.96.0/20"]
}

variable "vpc_id" {
  description = "VPC id, must be set correctly if create_vpc is false"
  type        = string
  default     = ""
}
