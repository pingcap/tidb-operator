variable "aks_resource_group" {
  description = "Azure resource group to run the cluster on"
}

variable "availability_zones" {
  description = "A list of Availability Zones where the Nodes in this Node Pool should be created in."
  type        = list(string)
}

variable "aks_cluster_id" {
  description = "AKS cluster ID. This module depends on a running cluster. Please create a cluster first and pass ID here."
}

variable "aks_subnet_id" {
  description = "Subnet ID for the AKS cluster"
}

variable "cluster_name" {
  description = "The TiDB cluster name"
}

variable "cluster_version" {
  description = "The TiDB cluster version"
  default     = "v4.0.10"
}

variable "kubeconfig_path" {}

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
