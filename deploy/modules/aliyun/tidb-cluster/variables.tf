variable "ack" {
  description = "The reference of the target ACK cluster"
}

variable "cluster_name" {
  description = "The TiDB cluster name"
}

variable "image_id" {
  default = "centos_7_06_64_20G_alibase_20190218.vhd"
}

variable "tidb_version" {
  description = "TiDB cluster version"
  default     = "v3.0.13"
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

variable "override_values" {
  type    = string
  default = ""
}

variable "local_exec_interpreter" {
  description = "Command to run for local-exec resources. Must be a shell-style interpreter. If you are on Windows Git Bash is a good choice."
  type        = list(string)
  default     = ["/bin/sh", "-c"]
}

variable "create_tidb_cluster_release" {
  description = "whether creating tidb-cluster helm release"
  default     = false
}
