variable "aks_cluster_id" {
  description = "AKS cluster ID. This module depends on a running cluster. Please create a cluster first and pass ID here."
}

variable "aks_subnet_id" {
  description = "Subnet ID for the AKS cluster"
}

variable "tidb_operator_id" {
  description = "TiDB Operator ID. We must wait for tidb-operator is ready before creating TiDB clusters."
}

variable "cluster_name" {}
variable "cluster_version" {
  description = "The TiDB cluster version"
  default     = "v4.0.10"
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
variable "aks_resource_group" {}
// variable "aks_cluster_name" {}
// variable "aks_cluster_location" {}
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
  default = "Standard_B2s"
}

variable "tikv_local_ssd_count" {
  description = "TiKV node pool local ssd count (cannot be changed after the node pool is created)"
  default     = 1
}

variable "create_tidb_cluster_release" {
  description = "Whether create tidb-cluster release in the node pools automatically"
  default     = true
}
