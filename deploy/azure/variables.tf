//variable "GCP_CREDENTIALS_PATH" {
//  description = "A path to to a service account key. See the docs for how to create one with the correct permissions"
//}

variable "REGION" {
  description = "The region in which to create the AKS cluster and associated resources"
  default = "westus2"
}

variable "RESOURCE_GROUP" {
  description = "The Azure resource group in which to create the necessary resources"
  default = "tidb-k8s"
}

variable "location" {
  description = "The AKS cluster location. If you specify a zone (such as us-central1-a), the cluster will be a zonal cluster with a single cluster master. If you specify a region (such as us-west1), the cluster will be a regional cluster with multiple masters spread across zones in the region. If not specified, the cluster will be a regional cluster in GCP_REGION."
  type        = string
  default     = ""
}

variable "node_locations" {
  description = "The list of zones in which the cluster's nodes should be located. These must be in the same region as the cluster zone for zonal clusters, or in the region of a regional cluster. In a multi-zonal cluster, the number of nodes specified in initial_node_count is created in all specified zones as well as the primary zone. If specified for a regional cluster, nodes will be created in only these zones."
  type        = list(string)
  default     = []
}

variable "tidb_version" {
  description = "TiDB version"
  default     = "v4.0.10"
}

variable "tidb_operator_version" {
  default = "v1.1.11"
}

variable "tidb_operator_chart_version" {
  description = "TiDB operator chart version, defaults to tidb_operator_version"
  default     = ""
}

variable "operator_helm_values" {
  description = "Operator helm values"
  type        = string
  default     = ""
}

variable "operator_helm_values_file" {
  description = "The helm values file for TiDB Operator, path is relative to current working dir"
  default     = ""
}

variable "create_vpc" {
  default = true
}

variable "aks_name" {
  description = "Name of the AKS cluster. Also used as a prefix in names of related resources."
  default     = "tidb-cluster"
}

variable "aks_version" {
  description = "Kubernetes version to use for the AKS cluster"
  type        = string
  default     = "1.19.7"
}

variable "default_tidb_cluster_name" {
  description = "The name that will be given to the default tidb cluster created."
  default     = "tidb-cluster"
}

variable "vpc_name" {
  description = "The name of the VPC network"
  default     = "tidb-cluster"
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
variable "pd_instance_type" {
  default = ""
}

variable "tikv_instance_type" {
  default = ""
}

variable "tidb_instance_type" {
  default = ""
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

variable "override_values" {
  description = "YAML formatted values that will be passed in to the tidb-cluster helm release"
  default     = ""
}

variable "override_values_file" {
  description = "The helm values file for TiDB Cluster, path is relative to current working dir"
  default     = ""
}

variable "create_tidb_cluster_release" {
  description = "whether creating tidb-cluster helm release"
  default     = false
}
