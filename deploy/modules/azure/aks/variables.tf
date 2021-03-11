variable "aks_name" {
  description = "Name of the AKS cluster. Also used as a prefix in names of related resources."
  type        = string
}

variable "aks_version" {
  description = "Kubernetes version to use for the AKS cluster"
  type        = string
  default     = "latest"
}

variable "region" {
  description = "The AKS cluster region. If you specify a zone (such as us-central1-a), the cluster will be a zonal cluster with a single cluster master. If you specify a region (such as us-west1), the cluster will be a regional cluster with multiple masters spread across zones in the region."
  type        = string
}

variable "resource_group" {
  description = "The resource group of this AKS cluster"
  type        = string
}

variable "vpc_name" {
  description = "Name of the VPC"
}

variable "ssh_key_data" {
  description = "SSH key for login to the cluster nodes"
  type        = string
}


# see https://docs.microsoft.com/en-us/azure/aks/uptime-sla
variable "aks_sku_tier" {
  description = "Kubernetes version to use for the GKE cluster"
  type        = string
  default     = "Free"
}

variable "availability_zones" {
  description = "The list of zones in which the cluster's nodes should be located. These must be in the same region as the cluster zone for zonal clusters, or in the region of a regional cluster. In a multi-zonal cluster, the number of nodes specified in initial_node_count is created in all specified zones as well as the primary zone. If specified for a regional cluster, nodes will be created in only these zones."
  type        = list(string)
}

variable "default_pool_node_name" {
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

variable "kubeconfig_path" {
  description = "kubeconfig path"
}
