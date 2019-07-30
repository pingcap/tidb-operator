variable "cluster_name" {}
variable "cluster_version" {}
variable "pd_replica_count" {}
variable "tikv_replica_count" {}
variable "tidb_replica_count" {}
variable "tidb_cluster_chart_version" {}
variable "override_values" {
  default = ""
}
variable "kubeconfig_path" {}