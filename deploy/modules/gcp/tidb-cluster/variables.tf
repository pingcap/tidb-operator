variable "cluster_name" {}
variable "cluster_version" {
  description = "The TiDB cluster version"
  default     = "v3.0.5"
}
variable "tidb_cluster_chart_version" {
  description = "The TiDB cluster chart version"
  default     = "v1.0.5"
}
variable "override_values" {
  description = "YAML formatted values that will be passed in to the tidb-cluster helm release"
  default     = ""
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
