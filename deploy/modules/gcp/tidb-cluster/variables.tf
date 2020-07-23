variable "cluster_id" {
  description = "GKE cluster ID. This module depends on a running cluster. Please create a cluster first and pass ID here."
}

variable "tidb_operator_id" {
  description = "TiDB Operator ID. We must wait for tidb-operator is ready before creating TiDB clusters."
}

variable "cluster_name" {}
variable "cluster_version" {
  description = "The TiDB cluster version"
  default     = "v4.0.0"
}
variable "tidb_cluster_chart_version" {
  description = "The TiDB cluster chart version"
  default     = "v1.0.6"
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

variable "pd_image_type" {
  description = "PD image type, available: UBUNTU/COS"
  default     = "COS"
}

variable "tidb_image_type" {
  description = "TiDB image type, available: UBUNTU/COS"
  default     = "COS"
}

variable "tikv_image_type" {
  description = "TiKV image type, available: UBUNTU/COS"
  default     = "COS"
}

variable "tikv_local_ssd_count" {
  description = "TiKV node pool local ssd count (cannot be changed after the node pool is created)"
  default     = 1
}

variable "create_tidb_cluster_release" {
  description = "Whether create tidb-cluster release in the node pools automatically"
  default     = true
}
