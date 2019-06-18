variable "cluster_name" {
  description = "TiDB clustername"
  default     = "tidb"
}

variable "tidb_version" {
  description = "TiDB version"
  default     = "v3.0.0-rc.1"
}

variable "tidb_operator_version" {
  description = "TiDB operator version"
  default     = "v1.0.0-beta.3"
}

variable "tidb_operator_registry" {
  description = "TiDB operator registry"
  default     = "pingcap"
}

variable "pd_replica_count" {
  default = 3
}

variable "tikv_replica_count" {
  default = 3
}

variable "tidb_replica_count" {
  default = 3
}

variable "pd_count" {
  description = "Number of PD nodes per availability zone"
  default     = 1
}

variable "tikv_count" {
  description = "Number of TiKV nodes per availability zone"
  default     = 1
}

variable "tidb_count" {
  description = "Number of TiDB nodes per availability zone"
  default     = 1
}

variable "monitor_count" {
  description = "Number of monitor nodes per availability zone"
  default     = 1
}
variable "pd_instance_type" { }

variable "tikv_instance_type" { }

variable "tidb_instance_type" { }

variable "monitor_instance_type" {
  default = "n1-standard-2"
}

variable "bastion_instance_type" {
  default = "f1-micro"
}

