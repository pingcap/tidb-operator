variable "kubeconfig_path" {
  description = "kubeconfig path"
}

variable "tidb_operator_version" {
  description = "TiDB Operator version"
  type        = string
  default     = "v1.1.11"
}

variable "operator_helm_values" {
  description = "Operator helm values"
  type        = string
  default     = ""
}
