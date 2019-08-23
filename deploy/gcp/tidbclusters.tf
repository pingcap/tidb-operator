provider "helm" {
  alias          = "gke"
  insecure       = true
  install_tiller = false
  kubernetes {
    config_path = local.kubeconfig
  }
}

module "default-tidb-cluster" {
  providers = {
    helm = "helm.gke"
  }
  source                     = "../modules/gcp/tidb-cluster"
  cluster_id                 = module.tidb-operator.cluster_id
  gcp_project                = var.GCP_PROJECT
  gke_cluster_location       = var.GCP_REGION
  gke_cluster_name           = var.gke_name
  cluster_name               = var.default_tidb_cluster_name
  cluster_version            = var.tidb_version
  kubeconfig_path            = local.kubeconfig
  tidb_cluster_chart_version = coalesce(var.tidb_operator_chart_version, var.tidb_operator_version)
  pd_instance_type           = var.pd_instance_type
  tikv_instance_type         = var.tikv_instance_type
  tidb_instance_type         = var.tidb_instance_type
  pd_image_type              = var.pd_image_type
  tikv_image_type            = var.tikv_image_type
  tidb_image_type            = var.tidb_image_type
  monitor_instance_type      = var.monitor_instance_type
  pd_node_count              = var.pd_count
  tikv_node_count            = var.tikv_count
  tidb_node_count            = var.tidb_count
  monitor_node_count         = var.monitor_count
  tikv_local_ssd_count       = var.tikv_local_ssd_count
  override_values            = var.override_values
}
