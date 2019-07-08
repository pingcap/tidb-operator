variable "ack" {
  description = "The reference of the target ACK cluster"
}

variable "cluster_name" {
  description = "The TiDB cluster name"
}

variable "key_name" {
  description = "The SSH key name for worker instances"
}

variable "group_id" {
  description = "The security group id for worker instances"
}

variable "tidb_version" {
  description = "TiDB cluster version"
  default     = "v3.0.0"
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
  default = "ecs.c5.4xlarge"
}

variable "monitor_instance_type" {
  description = "Monitor instance type"
  default = "ecs.c5.xlarge"
}
