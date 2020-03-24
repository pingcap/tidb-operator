module "tidb-cluster" {
  source = "../../share/tidb-cluster-release"

  cluster_name               = var.cluster_name
  cluster_version            = var.tidb_version
  pd_count                   = var.pd_count
  tikv_count                 = var.tikv_count
  tidb_count                 = var.tidb_count
  tidb_cluster_chart_version = var.tidb_cluster_chart_version
  override_values            = var.override_values
  local_exec_interpreter     = var.local_exec_interpreter
  base_values                = file("${path.module}/values/default.yaml")
  kubeconfig_filename        = var.ack.kubeconfig_filename
  service_ingress_key        = "ip"
  create                     = var.create_tidb_cluster_release
}
