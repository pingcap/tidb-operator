data "local_file" "kubeconfig" {
  depends_on = [module.tidb-operator.cluster_id]
  filename   = module.tidb-operator.kubeconfig_path
}
resource "local_file" "kubeconfig" {
  depends_on = [module.tidb-operator.cluster_id]
  content    = data.local_file.kubeconfig.content
  filename   = module.tidb-operator.kubeconfig_path
}

provider "helm" {
  alias          = "gke"
  insecure       = true
  install_tiller = false
  kubernetes {
    config_path = local_file.kubeconfig.filename
  }
}
module "default-tidb-cluster" {
  providers = {
    helm = "helm.gke"
  }
  source                     = "../modules/gcp/tidb-cluster"
  gcp_project                = module.tidb-operator.gcp_project
  gke_cluster_location       = module.tidb-operator.gke_cluster_location
  gke_cluster_name           = module.tidb-operator.gke_cluster_name
  cluster_name               = var.default_tidb_cluster_name
  cluster_version            = var.tidb_version
  kubeconfig_path            = module.tidb-operator.kubeconfig_path
  tidb_cluster_chart_version = var.tidb_operator_version
  pd_instance_type           = var.pd_instance_type
  tikv_instance_type         = var.tikv_instance_type
  tidb_instance_type         = var.tidb_instance_type
  monitor_instance_type      = var.monitor_instance_type
  pd_node_count              = var.pd_count
  tikv_node_count            = var.tikv_count
  tidb_node_count            = var.tidb_count
  monitor_node_count         = var.monitor_count
}
