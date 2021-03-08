output "vpc_name" {
  value = var.create_vpc ? azurerm_virtual_network.vpc_network[0].name : var.vpc_name
}
