# The name of a node pool may only contain lowercase alphanumeric characters and must begin with a lowercase letter.
# For Linux node pools the length must be between 1 and 12 characters
resource "azurerm_kubernetes_cluster_node_pool" "pd_pool" {
  name                  = "${var.cluster_name}-pd"
  kubernetes_cluster_id = var.aks_cluster_id
  vm_size               = var.pd_instance_type
  node_count            = var.pd_node_count
  vnet_subnet_id        = var.aks_subnet_id
  node_labels = {
    dedicated                            = "${var.cluster_name}-pd"
    "kubernetes.azure.com/aks-local-ssd" = true
  }
}

resource "azurerm_kubernetes_cluster_node_pool" "tidb_pool" {
  name                  = "${var.cluster_name}-tidb"
  kubernetes_cluster_id = var.aks_cluster_id
  vm_size               = var.tidb_instance_type
  node_count            = var.tidb_node_count
  vnet_subnet_id        = var.aks_subnet_id
  node_labels = {
    dedicated                            = "${var.cluster_name}-tidb"
  }
}

resource "azurerm_kubernetes_cluster_node_pool" "tikv_pool" {
  name                  = "${var.cluster_name}-tikv"
  kubernetes_cluster_id = var.aks_cluster_id
  vm_size               = var.tikv_instance_type
  node_count            = var.tikv_node_count
  vnet_subnet_id        = var.aks_subnet_id
  node_labels = {
    dedicated                            = "${var.cluster_name}-tikv"
    "kubernetes.azure.com/aks-local-ssd" = true
  }
}

resource "azurerm_kubernetes_cluster_node_pool" "monitor_pool" {
  // Setup local SSD on TiKV nodes first (this can take some time)
  // Create the monitor pool next because that is where tiller will be deployed to
  depends_on = [azurerm_kubernetes_cluster_node_pool.tikv_pool]
  name                  = "${var.cluster_name}-monitor-pool"
  kubernetes_cluster_id = var.aks_cluster_id
  vm_size               = var.monitor_instance_type
  node_count            = 3
  vnet_subnet_id        = var.aks_subnet_id
}

module "tidb-cluster" {
  source                     = "../../share/tidb-cluster-release2"

  create                     = var.create_tidb_cluster_release
  cluster_name               = var.cluster_name
  pd_count                   = var.pd_node_count
  tikv_count                 = var.tikv_node_count
  tidb_count                 = var.tidb_node_count
  tidb_cluster_chart_version = var.tidb_cluster_chart_version
  cluster_version            = var.cluster_version
  override_values            = var.override_values
  kubeconfig_filename        = var.kubeconfig_path
  base_values                = file("${path.module}/values/default.yaml")
  wait_on_resource           = [azurerm_kubernetes_cluster_node_pool.tidb_pool, var.tidb_operator_id]
  service_ingress_key        = "ip"
}

resource "null_resource" "wait-lb-ip" {
  count = var.create_tidb_cluster_release == true ? 1 : 0
  depends_on = [
    module.tidb-cluster
  ]
  provisioner "local-exec" {
    interpreter = ["bash", "-c"]
    working_dir = path.cwd
    command     = <<EOS
set -euo pipefail

until kubectl get svc -n ${var.cluster_name} ${var.cluster_name}-tidb -o json | jq '.status.loadBalancer.ingress[0]' | grep ip; do
  echo "Wait for TiDB internal loadbalancer IP"
  sleep 5
done
EOS

    environment = {
      KUBECONFIG = var.kubeconfig_path
    }
  }
}