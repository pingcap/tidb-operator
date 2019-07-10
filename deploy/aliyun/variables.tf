variable "ack_name" {
  description = "The name of alibaba container kubernetes cluster"
  default     = "my-cluster"
}

variable "operator_version" {
  description = "TiDB Operator version"
  type        = string
  default     = "v1.0.0-beta.3"
}

variable "vpc_id" {
  description = "VPC id, specify this variable to use an exsiting VPC and the vswitches in the VPC. Note that when using existing vpc, it is recommended to use a existing security group too. Otherwise you have to set vpc_cidr according to the existing VPC settings to get correct in-cluster security rule."
  default     = ""
}

variable "group_id" {
  description = "Security group id, specify this variable to use and exising security group"
  default     = ""
}

variable "vpc_cidr_newbits" {
  description = "VPC cidr newbits, it's better to be set as 16 if you use 10.0.0.0/8 cidr block"
  default     = "8"
}

variable "k8s_pod_cidr" {
  description = "The kubernetes pod cidr block. It cannot be equals to vpc's or vswitch's and cannot be in them. Cannot change once the cluster created."
  default     = "172.20.0.0/16"
}

variable "k8s_service_cidr" {
  description = "The kubernetes service cidr block. It cannot be equals to vpc's or vswitch's or pod's and cannot be in them. Cannot change once the cluster created."
  default     = "172.21.0.0/20"
}

variable "vpc_cidr" {
  description = "VPC cidr_block, options: [192.168.0.0.0/16, 172.16.0.0/16, 10.0.0.0/8], cannot collidate with kubernetes service cidr and pod cidr. Cannot change once the vpc created."
  default     = "192.168.0.0/16"
}

variable "create_bastion" {
  description = "Whether create bastion server"
  default     = true
}

variable "bastion_image_name" {
  description = "OS image of bastion"
  default     = "centos_7_06_64_20G_alibase_20190218.vhd"
}

variable "bastion_key_prefix" {
  default = "bastion-key"
}

variable "bastion_cpu_core_count" {
  description = "CPU core count to select bastion type"
  default     = 1
}

variable "bastion_ingress_cidr" {
  description = "Bastion ingress security rule cidr, it is highly recommended to set this in favor of safety"
  default     = "0.0.0.0/0"
}
