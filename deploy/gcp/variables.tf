variable "cluster_name" {
  description = "TiDB clustername"
  default = "tidb-cluster"
}

variable "tidb_version" {
  description = "TiDB version"
  default = "v2.1.8"
}

variable "pd_count" {
  description = "Number of PD nodes per availability zone"
  default = 1
}

variable "tikv_count" {
  description = "Number of TiKV nodes per availability zone"
  default = 1
}

variable "tidb_count" {
  description = "Number of TiDB nodes per availability zone"
  default = 1
}

variable "pd_instance_type" {
  default = "n1-standard-1"
}

variable "tikv_instance_type" {
  default = "n1-standard-1"
}

variable "tidb_instance_type" {
  default = "n1-standard-1"
}

variable "monitor_instance_type" {
  default = "n1-standard-1"
}
