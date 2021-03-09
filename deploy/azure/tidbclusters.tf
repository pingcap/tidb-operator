module "default-tidb-cluster" {
  source                      = "../modules/azure/tidb-cluster"

  # aks
  aks_cluster_id              = module.aks.aks_cluster_id
  aks_resource_group          = var.resource_group
  # aks_cluster_location        = local.location
  aks_subnet_id               = module.aks.aks_subnet_id
  kubeconfig_path             = local.kubeconfig_path

  # tidb operator
  tidb_operator_id            = module.tidb-operator.tidb_operator_id
  tidb_cluster_chart_version  = coalesce(var.tidb_operator_chart_version, var.tidb_operator_version)

  # tidb
  cluster_name                = var.tidb_cluster_name
  cluster_version             = var.tidb_version
  pd_instance_type            = var.pd_instance_type
  tikv_instance_type          = var.tikv_instance_type
  tidb_instance_type          = var.tidb_instance_type
  monitor_instance_type       = var.monitor_instance_type
  pd_node_count               = var.pd_count
  tikv_node_count             = var.tikv_count
  tidb_node_count             = var.tidb_count
  monitor_node_count          = var.monitor_count
  override_values             = var.override_values == "" ? var.override_values_file == "" ? "" : file(var.override_values_file) : var.override_values
  create_tidb_cluster_release = var.create_tidb_cluster_release
}


