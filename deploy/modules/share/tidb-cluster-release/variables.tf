variable "kubeconfig_filename" {
  description = "The kubeconfig filename, path should be relative to current working dir"
  default = ""
}

variable "tidb_cluster_chart_version" {
  description = "tidb-cluster chart version"
  default     = "v1.0.0-beta.3"
}

variable "cluster_name" {
  type        = string
  description = "tidb cluster name"
}

variable "cluster_version" {
  type    = string
  default = "v3.0.1"
}

variable "pd_count" {
  type    = number
  default = 3
}

variable "tikv_count" {
  type    = number
  default = 3
}

variable "tidb_count" {
  type    = number
  default = 2
}

variable "base_values" {
  type = string
  default = ""
}

variable "override_values" {
  type    = string
  default = ""
}

variable "service_ingress_key" {
  type    = string
  default = "hostname"
}

variable "local_exec_interpreter" {
  description = "Command to run for local-exec resources. Must be a shell-style interpreter. If you are on Windows Git Bash is a good choice."
  type        = list(string)
  default     = ["/bin/sh", "-c"]
}
