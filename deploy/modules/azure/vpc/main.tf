resource "azurerm_virtual_network" "vpc_network" {
  count               = var.create_vpc ? 1 : 0
  name                = var.vpc_name
  address_space       = var.vpc_cidr
  location            = var.region
  resource_group_name = var.resource_group
}