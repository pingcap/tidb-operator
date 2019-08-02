variable "vpc_name" {
  description = "Name of the VPC"
}

variable "create_vpc" {
  description = "Create a new VPC or not"
}

variable "private_subnet_primary_cidr_range" {
  description = "Primary CIDR range for the private subnetwork"
  default     = "172.31.252.0/22"
}

variable "private_subnet_secondary_cidr_ranges" {
  description = "Secondary ranges for private subnet"
  default     = ["172.30.0.0/16", "172.31.224.0/20"]
}

variable "private_subnet_name" {
  description = "Name for the private subnet"
  default     = "private-subnet"
}

variable "public_subnet_name" {
  description = "Name for the public subnet"
  default     = "public-subnet"
}

variable "public_subnet_primary_cidr_range" {
  description = "Primary CIDR range for the public subnetwork"
  default     = "172.29.252.0/22"
}

variable "gcp_region" {
  description = "The GCP region name"
}

variable "gcp_project" {
  description = "The GCP project name"
}
