module "default-tidb-cluster" {
  source                      = "../modules/azure/tidb-cluster"
  aks_cluster_id              = module.aks.aks_cluster_id
  tidb_operator_id            = module.tidb-operator.tidb_operator_id
  aks_resource_group          = var.RESOURCE_GROUP
  aks_cluster_location        = local.location
  aks_subnet_id               = module.aks.aks_subnet_id
  cluster_name                = var.default_tidb_cluster_name
  cluster_version             = var.tidb_version
  kubeconfig_path             = local.kubeconfig_path
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


