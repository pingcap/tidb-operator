provider "helm" {
  alias          = "gke"
  insecure       = true
  install_tiller = false
  kubernetes {
    # helm provider loads the file when it's initialized, we must wait for it to be created.
    # However we cannot use resource here, because in refresh phrase, it will
    # not be resolved and argument default value is used. To work around this,
    # we defer initialization by using load_config_file argument.
    # See https://github.com/pingcap/tidb-operator/pull/819#issuecomment-524547459
    config_path = local.kubeconfig
    # used to delay helm provisioner initialization in apply phrase
    load_config_file = module.tidb-operator.get_credentials_id != "" ? true : null
  }
}

module "default-tidb-cluster" {
  providers = {
    helm = "helm.gke"
  }
  source                      = "../modules/gcp/tidb-cluster"
  cluster_id                  = module.tidb-operator.cluster_id
  tidb_operator_id            = module.tidb-operator.tidb_operator_id
  gcp_project                 = var.GCP_PROJECT
  gke_cluster_location        = local.location
  gke_cluster_name            = var.gke_name
  cluster_name                = var.default_tidb_cluster_name
  cluster_version             = var.tidb_version
  kubeconfig_path             = local.kubeconfig
  tidb_cluster_chart_version  = coalesce(var.tidb_operator_chart_version, var.tidb_operator_version)
  pd_instance_type            = var.pd_instance_type
  tikv_instance_type          = var.tikv_instance_type
  tidb_instance_type          = var.tidb_instance_type
  pd_image_type               = var.pd_image_type
  tikv_image_type             = var.tikv_image_type
  tidb_image_type             = var.tidb_image_type
  monitor_instance_type       = var.monitor_instance_type
  pd_node_count               = var.pd_count
  tikv_node_count             = var.tikv_count
  tidb_node_count             = var.tidb_count
  monitor_node_count          = var.monitor_count
  tikv_local_ssd_count        = var.tikv_local_ssd_count
  override_values             = var.override_values == "" ? var.override_values_file == "" ? "" : file(var.override_values_file) : var.override_values
  create_tidb_cluster_release = var.create_tidb_cluster_release
}
