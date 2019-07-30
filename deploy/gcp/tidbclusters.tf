provider "helm" {
  alias = "gke"
  insecure = true
  install_tiller = false
  kubernetes {
    config_path = local.kubeconfig
  }
}
//module "default-tidb-cluster" {
//  source = "../modules/gcp/tidb-cluster"
//  cluster_name = var.gke_name
//  cluster_version = var.tidb_version
//  kubeconfig_path = local.kubeconfig
//  tidb_cluster_chart_version = var.tidb_operator_version
//  pd_replica_count = var.pd_replica_count
//  tikv_replica_count = var.tikv_replica_count
//  tidb_replica_count = var.tidb_replica_count
//}