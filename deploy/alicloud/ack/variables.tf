variable "cluster_name" {
  description = "Kubernetes cluster name"
  default     = "tidb_cluster"
}

variable "vpc_cidr" {
  description = "VPC cidr_block, options: [192.168.0.0.0/16, 172.16.0.0/16, 10.0.0.0/8], cannot collidate with kubernetes service cidr and pod cidr"
  default     = "192.168.0.0/16"
}

variable "vpc_id" {
  description = "VPC id, specify this variable to use an exiting VPC and the vswitches in the VPC"
  default     = ""
}

variable "security_group_id" {
  description = "Security group of ACK cluster"
  default     = ""
}

variable "vpc_cidr_newbits" {
  description = "VPC cidr newbits, it's better to be set as 16 if you use 10.0.0.0/8 cidr block"
  default     = "8"
}

variable "k8s_pod_cidr" {
  description = "The kubernetes pod cidr block. It cannot be equals to vpc's or vswitch's and cannot be in them."
  default     = "172.20.0.0/16"
}

variable "k8s_service_cidr" {
  description = "The kubernetes service cidr block. It cannot be equals to vpc's or vswitch's or pod's and cannot be in them."
  default     = "172.21.0.0/20"
}

variable "create_nat_gateway" {
  description = "If create nat gateway in VPC"
  default     = true
}

variable "cluster_network_type" {
  description = "Kubernetes network plugin, options: [flannel, terway]"
  default     = "flannel"
}

variable "public_apiserver" {
  description = "Whether enable apiserver internet access"
  default     = false
}

variable "worker_groups" {
  description = "A list of maps defining worker group configurations to be defined using alicloud ESS. See group_default for validate keys"
  type        = "list"

  default = [
    {
      "name" = "default"
    },
  ]
}

variable "group_default" {
  description = "The default values for all worker groups"
  type        = "map"

  default = {
    "min_size"             = 0
    "max_size"             = 100
    "default_cooldown"     = 300
    "multi_az_policy"      = "BALANCE"
    "image_id"             = "centos_7_06_64_20G_alibase_20190218.vhd"
    "instance_type"        = "ecs.g5.large"
    "system_disk_category" = "cloud_efficiency"
    "system_disk_size"     = 50
    "data_disk"            = {}
    "pre_userdata"         = ""
    "post_userdata"        = ""
    "autoscaling_enabled"  = false
  }
}

variable "group_tags" {
  description = "A map of tags to add to ecs instances."
  type        = "map"
  default     = {}
}
