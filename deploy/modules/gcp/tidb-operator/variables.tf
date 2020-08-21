variable "gke_name" {
  description = "Name of the GKE cluster. Also used as a prefix in names of related resources."
  type        = string
}

variable "vpc_name" {
  description = "The name of the VPC in which to place the cluster"
}

variable "subnetwork_name" {
  description = "The name of the subnetwork in which to place the cluster"
}

variable "gcp_project" {
  description = "The GCP project name"
}

variable "location" {
  description = "The GKE cluster location. If you specify a zone (such as us-central1-a), the cluster will be a zonal cluster with a single cluster master. If you specify a region (such as us-west1), the cluster will be a regional cluster with multiple masters spread across zones in the region."
  type        = string
}

variable "node_locations" {
  description = "The list of zones in which the cluster's nodes should be located. These must be in the same region as the cluster zone for zonal clusters, or in the region of a regional cluster. In a multi-zonal cluster, the number of nodes specified in initial_node_count is created in all specified zones as well as the primary zone. If specified for a regional cluster, nodes will be created in only these zones."
  type        = list(string)
}

variable "gke_version" {
  description = "Kubernetes version to use for the GKE cluster"
  type        = string
  default     = "latest"
}

variable "tidb_operator_version" {
  description = "TiDB Operator version"
  type        = string
  default     = "v1.0.6"
}

variable "operator_helm_values" {
  description = "Operator helm values"
  type        = string
  default     = ""
}

variable "kubeconfig_path" {
  description = "kubeconfig path"
}

variable "maintenance_window_start_time" {
  description = "The time in HH:MM GMT format to define the start of the daily maintenance window"
  default     = "01:00"
}
