variable "region" {
  description = "AWS region"
}

variable "eks_name" {
  description = "Name of the EKS cluster. Also used as a prefix in names of related resources."
  type        = string
}

variable "eks_version" {
  description = "Kubernetes version to use for the EKS cluster."
  type        = string
  default     = "1.12"
}

variable "operator_version" {
  description = "TiDB Operator version"
  type        = string
  default     = "v1.0.6"
}

variable "operator_helm_values" {
  description = "Operator helm values"
  type        = string
  default     = ""
}

variable "config_output_path" {
  description = "Where to save the Kubectl config file (if `write_kubeconfig = true`). Should end in a forward slash `/` ."
  type        = string
  default     = "./"
}

variable "subnets" {
  description = "A list of subnets to place the EKS cluster and workers within."
  type        = list(string)
}

variable "vpc_id" {
  description = "VPC where the cluster and workers will be deployed."
  type        = string
}

variable "default_worker_group_instance_type" {
  description = "The instance type of default worker groups, this group will be used to run tidb-operator"
  default     = "m4.large"
}

variable "default_worker_group_instance_count" {
  description = "The instance count of default worker groups, this group will be used to run tidb-operator"
  default     = 1
}

variable "ssh_key_name" {
  type = string
}
