output "aks_cluster_id" {
  value = azurerm_kubernetes_cluster.cluster.id
}

output "aks_subnet_id" {
  value = azurerm_kubernetes_cluster.cluster.id
}

output "client_certificate" {
  value = azurerm_kubernetes_cluster.cluster.kube_config.0.client_certificate
}

output "kube_config" {
  value = azurerm_kubernetes_cluster.cluster.kube_config_raw
}