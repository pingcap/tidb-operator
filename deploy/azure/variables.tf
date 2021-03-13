variable "region" {
  description = "The region in which to create the AKS cluster and associated resources"
  default     = "westus2"
}

variable "resource_group" {
  description = "The Azure resource group in which to create the necessary resources"
  default     = "tidb"
}

variable "availability_zones" {
  description = "The list of zones in which the cluster's nodes should be located. These must be in the same region as the cluster zone for zonal clusters, or in the region of a regional cluster. In a multi-zonal cluster, the number of nodes specified in initial_node_count is created in all specified zones as well as the primary zone. If specified for a regional cluster, nodes will be created in only these zones."
  type        = list(string)
  default     = []
}


########################################### VPC ############################################
variable "create_vpc" {
  default = true
}

variable "vpc_name" {
  description = "The name of the VPC network"
  default     = "tidb-vpc"
}

########################################### AKS ############################################

variable "aks_name" {
  description = "Name of the AKS cluster. Also used as a prefix in names of related resources."
  default     = "aks"
}

variable "aks_version" {
  description = "Kubernetes version to use for the AKS cluster"
  type        = string
  default     = "1.19.7"
}

# see https://docs.microsoft.com/en-us/azure/aks/uptime-sla
variable "aks_sku_tier" {
  description = "Uptime SLA for the AKS cluster"
  type        = string
  default     = "Free"
}

variable "default_pool_name" {
  description = "Name of the default node pool"
  default     = "default"
}

variable "default_pool_node_count" {
  description = "Number of nodes in default node pool"
  default     = 1
}

variable "default_pool_instance_type" {
  description = "VM type of default node pool"
  default = "Standard_B2s"
}

variable "dns_service_ip" {
  description = ""
  default     = "10.0.0.10"
}

variable "docker_bridge_cidr" {
  description = ""
  default     = "172.17.0.1/16"
}

variable "service_cidr" {
  description = "VPC private subnets, must be set correctly if create_vpc is true"
  default     = "10.0.0.0/20"
}

variable "aks_cidr" {
  description = "VPC private subnets, must be set correctly if create_vpc is true"
  type        = list(string)
  default     = ["10.0.16.0/20"]
}

####################################### TiDB Operator #######################################
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

########################################### TiDB ###########################################

variable "tidb_cluster_name" {
  description = "The name that will be given to the default tidb cluster created."
  default     = "tidb"
}

variable "tidb_version" {
  description = "TiDB version"
  default     = "v4.0.10"
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

# The VM SKUs chosen for agentpool are restricted by AKS. Please see https://aka.ms/aks/restricted-skus for more details
variable "pd_instance_type" {
  default = "Standard_B2s"
}

variable "tikv_instance_type" {
  default = "Standard_B2s"
}

variable "tidb_instance_type" {
  default = "Standard_B2s"
}

variable "monitor_instance_type" {
  default = "Standard_B2s"
}

variable "bastion_instance_type" {
  default = "Standard_B1s"
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
