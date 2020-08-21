variable "vpc_name" {
  description = "The VPC network name"
}

variable "gcp_project" {}

variable "bastion_instance_type" {
  default = "f1-micro"
}

variable "public_subnet_name" {}

variable "bastion_name" {}
