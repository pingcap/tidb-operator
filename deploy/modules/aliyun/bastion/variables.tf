variable "bastion_image_name" {
  description = "OS image of bastion"
  default     = "centos_7_06_64_20G_alibase_20190218.vhd"
}

variable "bastion_cpu_core_count" {
  description = "CPU core count to select bastion type"
  default     = 1
}

variable "bastion_ingress_cidr" {
  description = "Bastion ingress security rule cidr, it is highly recommended to set this in favor of safety"
  default     = "0.0.0.0/0"
}

variable "key_name" {
  description = "bastion key name"
}

variable "vpc_id" {
  description = "VPC id"
}

variable "bastion_name" {
  description = "bastion name"
}

variable "vswitch_id" {
  description = "vswitch id"
}

variable "worker_security_group_id" {
  description = "The security group id of worker nodes, must be provided if enable_ssh_to_worker set to true"
  default     = ""
}

variable "enable_ssh_to_worker" {
  description = "Whether enable ssh connection from bastion to ACK workers"
  default     = false
}
