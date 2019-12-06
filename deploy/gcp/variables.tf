variable "GCP_CREDENTIALS_PATH" {
  description = "A path to to a service account key. See the docs for how to create one with the correct permissions"
}

variable "GCP_REGION" {
  description = "The GCP region in which to create the GKE cluster and associated resources"
}

variable "GCP_PROJECT" {
  description = "The GCP project in which to create the necessary resources"
}

variable "tidb_version" {
  description = "TiDB version"
  default     = "v3.0.5"
}

variable "tidb_operator_version" {
  default = "v1.0.5"
}

variable "tidb_operator_chart_version" {
  description = "TiDB operator chart version, defaults to tidb_operator_version"
  default     = ""
}

variable "create_vpc" {
  default = true
}

variable "gke_name" {
  description = "Name of the GKE cluster. Also used as a prefix in names of related resources."
  default     = "tidb-cluster"
}

variable "default_tidb_cluster_name" {
  description = "The name that will be given to the default tidb cluster created."
  default     = "tidb-cluster"
}

variable "vpc_name" {
  description = "The name of the VPC network"
  default     = "tidb-cluster"
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
variable "pd_instance_type" {}

variable "tikv_instance_type" {}

variable "tidb_instance_type" {}

variable "monitor_instance_type" {
  default = "n1-standard-2"
}

variable "bastion_instance_type" {
  default = "f1-micro"
}

variable "maintenance_window_start_time" {
  description = "The time in HH:MM GMT format to define the start of the daily maintenance window"
  default     = "01:00"
}
