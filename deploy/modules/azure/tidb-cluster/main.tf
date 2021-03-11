resource "azurerm_kubernetes_cluster_node_pool" "pd_pool" {
  # pool name must start with a lowercase letter, have max length of 12, and only have characters a-z0-9
  name                  = replace(lower("${var.cluster_name}pd"), "/[^\\w]/", "")
  kubernetes_cluster_id = var.aks_cluster_id
  availability_zones    = var.availability_zones
  vm_size               = var.pd_instance_type
  node_count            = var.pd_node_count
  vnet_subnet_id        = var.aks_subnet_id
  os_disk_size_gb       = 50
  node_taints           = ["dedicated=${var.cluster_name}-pd:NoSchedule"]
  node_labels = {
    dedicated                            = "${var.cluster_name}-pd"
    "kubernetes.azure.com/aks-local-ssd" = true
  }

}

resource "azurerm_kubernetes_cluster_node_pool" "tidb_pool" {
  # pool name must start with a lowercase letter, have max length of 12, and only have characters a-z0-9
  name                  = replace(lower("${var.cluster_name}db"), "/[^\\w]/", "")
  kubernetes_cluster_id = var.aks_cluster_id
  availability_zones    = var.availability_zones
  vm_size               = var.tidb_instance_type
  node_count            = var.tidb_node_count
  vnet_subnet_id        = var.aks_subnet_id
  os_disk_size_gb       = 50
  node_taints           = ["dedicated=${var.cluster_name}-tidb:NoSchedule"]
  node_labels = {
    dedicated                            = "${var.cluster_name}-tidb"
  }
}

resource "azurerm_kubernetes_cluster_node_pool" "tikv_pool" {
  # pool name must start with a lowercase letter, have max length of 12, and only have characters a-z0-9
  name                  = replace(lower("${var.cluster_name}kv"), "/[^\\w]/", "")
  kubernetes_cluster_id = var.aks_cluster_id
  availability_zones    = var.availability_zones
  vm_size               = var.tikv_instance_type
  node_count            = var.tikv_node_count
  vnet_subnet_id        = var.aks_subnet_id
  os_disk_size_gb       = 50
  node_taints           = ["dedicated=${var.cluster_name}-tikv:NoSchedule"]
  node_labels = {
    dedicated                            = "${var.cluster_name}-tikv"
    "kubernetes.azure.com/aks-local-ssd" = true
  }
}

resource "azurerm_kubernetes_cluster_node_pool" "monitor_pool" {
  // Setup local SSD on TiKV nodes first (this can take some time)
  depends_on = [azurerm_kubernetes_cluster_node_pool.tikv_pool]
  name                  = replace(lower("${var.cluster_name}mo"), "/[^\\w]/", "")
  kubernetes_cluster_id = var.aks_cluster_id
  vm_size               = var.monitor_instance_type
  node_count            = var.monitor_node_count
  vnet_subnet_id        = var.aks_subnet_id
}


//resource "null_resource" "setup-operator" {
//  provisioner "local-exec" {
//    command = "kubectl apply -f https://raw.githubusercontent.com/pingcap/tidb-operator/${var.tidb_operator_version}/manifests/crd.yaml"
//    environment = {
//      KUBECONFIG = var.kubeconfig_path
//    }
//  }
//}

//module "tidb-cluster" {
//  source                     = "../../share/tidb-cluster-release2"
//
//  create                     = var.create_tidb_cluster_release
//  cluster_name               = var.cluster_name
//  pd_count                   = var.pd_node_count
//  tikv_count                 = var.tikv_node_count
//  tidb_count                 = var.tidb_node_count
//  tidb_cluster_chart_version = var.tidb_cluster_chart_version
//  cluster_version            = var.cluster_version
//  override_values            = var.override_values
//  kubeconfig_filename        = var.kubeconfig_path
//  base_values                = file("${path.module}/values/default.yaml")
//  wait_on_resource           = [azurerm_kubernetes_cluster_node_pool.tidb_pool, var.tidb_operator_id]
//  service_ingress_key        = "ip"
//}
//
resource "null_resource" "wait-lb-ip" {
  count = var.create_tidb_cluster_release == true ? 1 : 0
  depends_on = [
    #module.tidb-cluster
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