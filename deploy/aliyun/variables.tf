variable "cluster_name_prefix" {
  description = "TiDB cluster name"
  default     = "tidb-cluster"
}

variable "tidb_version" {
  description = "TiDB cluster version"
  default     = "v3.0.0-rc.1"
}

variable "pd_count" {
  description = "PD instance count, the recommend value is 3"
  default     = 3
}

variable "pd_instance_type_family" {
  description = "PD instance type family, values: [ecs.i2, ecs.i1, ecs.i2g]"
  default     = "ecs.i2"
}

variable "pd_instance_memory_size" {
  description = "PD instance memory size in GB, must available in the type famliy"
  default     = 32
}

variable "tikv_count" {
  description = "TiKV instance count, ranges: [3, 100]"
  default     = 3
}

variable "tikv_instance_type_family" {
  description = "TiKV instance memory in GB, must available in type family"
  default     = "ecs.i2"
}

variable "tikv_memory_size" {
  description = "TiKV instance memory in GB, must available in type family"
  default     = 64
}

variable "tidb_count" {
  description = "TiDB instance count, ranges: [1, 100]"
  default     = 2
}

variable "tidb_instance_type" {
  description = "TiDB instance type, this variable override tidb_instance_core_count and tidb_instance_memory_size, is recommended to use the tidb_instance_core_count and tidb_instance_memory_size to select instance type in favor of flexibility"

  default = ""
}

variable "tidb_instance_core_count" {
  default = 16
}

variable "tidb_instance_memory_size" {
  default = 32
}

variable "monitor_intance_type" {
  description = "Monitor instance type, this variable override tidb_instance_core_count and tidb_instance_memory_size, is recommended to use the tidb_instance_core_count and tidb_instance_memory_size to select instance type in favor of flexibility"

  default = ""
}

variable "monitor_instance_core_count" {
  default = 4
}

variable "monitor_instance_memory_size" {
  default = 8
}

variable "monitor_storage_class" {
  description = "Monitor PV storageClass, values: [alicloud-disk-commo, alicloud-disk-efficiency, alicloud-disk-ssd, alicloud-disk-available]"
  default     = "alicloud-disk-available"
}

variable "monitor_storage_size" {
  description = "Monitor storage size in Gi"
  default     = 500
}

variable "monitor_reserve_days" {
  description = "Monitor data reserve days"
  default     = 14
}

variable "default_worker_core_count" {
  description = "CPU core count of default kubernetes workers"
  default     = 2
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

variable "monitor_slb_network_type" {
  description = "The monitor slb network type, values: [internet, intranet]. It is recommended to set it as intranet and access via VPN in favor of safety"
  default     = "internet"
}

variable "monitor_enable_anonymous_user" {
  description = "Whether enabling anonymous user visiting for monitoring"
  default     = false
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

