/** subnet to be used by kubernetes */
resource "azurerm_subnet" "aks_subnet" {
  name                 = "${var.aks_name}-subnet"
  address_prefixes     = var.aks_cidr
  resource_group_name  = var.resource_group
  virtual_network_name = var.vpc_name
}

resource "azurerm_kubernetes_cluster" "cluster" {
  depends_on           = [azurerm_subnet.aks_subnet]
  name                 = var.aks_name
  location             = var.region
  resource_group_name  = var.resource_group
  # The dns_prefix must contain between 3 and 45 characters, and can contain only letters, numbers, and hyphens.
  # It must start with a letter and must end with a letter or a number.
  dns_prefix           = substr(replace(join("-", [var.aks_name, var.resource_group]), "/[^\\w\\-]/", ""), 0, 45)
  kubernetes_version   = var.aks_version
  sku_tier             = var.aks_sku_tier

  network_profile {
    # hard-coded to azure CNI for better performance
    # however the pod ip addresses might conflict with node addresses with azure CNI, the ip addressed
    # must be planned in properly.
    # see https://docs.microsoft.com/en-us/azure/aks/configure-azure-cni for more details
    network_plugin     = "azure"
    dns_service_ip     = var.dns_service_ip
    docker_bridge_cidr = var.docker_bridge_cidr
    service_cidr       = var.service_cidr
  }

  # The default node pool is primary used for hosting critical system pods such as coredns and metrics-server
  default_node_pool {
    name               = var.default_pool_node_name
    node_count         = var.default_pool_node_count
    vm_size            = var.default_pool_instance_type
    vnet_subnet_id     = azurerm_subnet.aks_subnet.id
    type               = "VirtualMachineScaleSets"
  }

  identity {
    type = "SystemAssigned"
  }

}

resource "local_file" "kubeconfig" {
  depends_on          = [azurerm_kubernetes_cluster.cluster]
  sensitive_content   = azurerm_kubernetes_cluster.cluster.kube_config_raw
  filename            = var.kubeconfig_path
}

# aks initialization, right now it just adds local-storage.
resource "null_resource" "init" {
  depends_on          = [azurerm_kubernetes_cluster.cluster]

  provisioner "local-exec" {
    command           = "kubectl apply -f ${path.module}/manifests/local-ssd-provision.yaml"
    environment       = {
      KUBECONFIG      = var.kubeconfig_path
    }
  }
}