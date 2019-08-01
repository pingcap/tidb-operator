variable "cluster_name" {}
variable "cluster_version" {}
variable "pd_replica_count" {}
variable "tikv_replica_count" {}
variable "tidb_replica_count" {}
variable "tidb_cluster_chart_version" {}
variable "override_values" {
  default = ""
}
variable "kubeconfig_path" {}
variable "gcp_project" {}
variable "gke_cluster_name" {}
variable "gke_cluster_location" {}
variable "pd_node_count" {
  description = "Number of PD nodes per availability zone"
  default     = 1
}

variable "tikv_node_count" {
  description = "Number of TiKV nodes per availability zone"
  default     = 1
}

variable "tidb_node_count" {
  description = "Number of TiDB nodes per availability zone"
  default     = 1
}

variable "monitor_node_count" {
  description = "Number of monitor nodes per availability zone"
  default     = 1
}

variable "pd_instance_type" {}

variable "tikv_instance_type" {}

variable "tidb_instance_type" {}

variable "monitor_instance_type" {
  default = "n1-standard-2"
}
