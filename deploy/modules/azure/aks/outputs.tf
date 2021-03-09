output "aks_cluster_id" {
  value = azurerm_kubernetes_cluster.cluster.id
}

output "aks_subnet_id" {
  value = azurerm_subnet.aks_subnet.id
}

output "kubeconfig_path" {
  value = local_file.kubeconfig.filename
}

