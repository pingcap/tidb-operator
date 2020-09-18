variable "bastion_image_name" {
  description = "OS image of bastion"
  default     = "centos_7_06_64_20G_alibase_20190218.vhd"
}

variable "bastion_cpu_core_count" {
  description = "CPU core count to select bastion type"
  default     = 1
}

variable "operator_version" {
  type    = string
  default = "v1.1.5"
}

variable "operator_helm_values" {
  description = "The helm values file for TiDB Operator, path is relative to current working dir"
  type        = string
  default     = ""
}

variable "override_values" {
  description = "The helm values file for TiDB Cluster, path is relative to current working dir"
  default     = ""
}

variable "bastion_ingress_cidr" {
  description = "Bastion ingress security rule cidr, it is highly recommended to set this in favor of safety"
  default     = "0.0.0.0/0"
}

variable "cluster_name" {
  description = "Kubernetes cluster name"
  default     = "my-cluster"
}

variable "tidb_version" {
  description = "TiDB cluster version"
  default     = "v4.0.6"
}
variable "tidb_cluster_chart_version" {
  description = "tidb-cluster chart version"
  default     = "v1.0.6"
}

variable "pd_count" {
  description = "PD instance count, the recommend value is 3"
  default     = 3
}

variable "pd_instance_type" {
  description = "PD instance type"
  default     = "ecs.g5.large"
}

variable "tikv_count" {
  description = "TiKV instance count, ranges: [3, 100]"
  default     = 3
}

variable "tikv_instance_type" {
  description = "TiKV instance memory in GB, must available in type family"
  default     = "ecs.i2.2xlarge"
}

variable "tidb_count" {
  description = "TiDB instance count, ranges: [1, 100]"
  default     = 2
}

variable "tidb_instance_type" {
  description = "TiDB instance type"
  default     = "ecs.c5.4xlarge"
}

variable "monitor_instance_type" {
  description = "Monitor instance type"
  default     = "ecs.c5.xlarge"
}

variable "default_worker_core_count" {
  description = "CPU core count of default kubernetes workers"
  default     = 2
}

variable "vpc_id" {
  description = "VPC id"
  default     = ""
}

variable "group_id" {
  description = "Security group id, specify this variable to use and existing security group"
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

variable "create_tidb_cluster_release" {
  description = "whether creating tidb-cluster helm release"
  default     = false
}

variable "tidb_cluster_name" {
  description = "The TiDB cluster name"
  default     = "my-cluster"
}

variable "create_tiflash_node_pool" {
  description = "whether creating node pool for tiflash"
  default     = false
}

variable "create_cdc_node_pool" {
  description = "whether creating node pool for cdc"
  default     = false
}

variable "tiflash_count" {
  default = 2
}

variable "cdc_count" {
  default = 3
}

variable "cdc_instance_type" {
  default = "ecs.c5.2xlarge"
}

variable "tiflash_instance_type" {
  default = "ecs.i2.2xlarge"
}
