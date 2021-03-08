locals {
  # Create a regional cluster in current region by default.
  location                       = var.location != "" ? var.location : var.REGION
  credential_path                = "${path.cwd}/credentials"
  kubeconfig_path                = "${local.credential_path}/kubeconfig_${var.aks_name}"
}


provider "azurerm" {
  features {}
}

module "vpc" {
  source                          = "../modules/azure/vpc"

  create_vpc                      = var.create_vpc
  resource_group                  = var.RESOURCE_GROUP
  region                          = var.REGION
  vpc_name                        = var.vpc_name
}

module "aks" {
  source                          = "../modules/azure/aks"

  aks_name                        = var.aks_name
  aks_version                     = var.aks_version
  region                          = var.REGION
  resource_group                  = var.RESOURCE_GROUP
  vpc_name                        = var.vpc_name
  node_locations                  = var.node_locations
  kubeconfig_path                 = local.kubeconfig_path
}

module "tidb-operator" {
  source                        = "../modules/azure/tidb-operator"

  kubeconfig_path               = local.kubeconfig_path
  tidb_operator_version         = var.tidb_operator_version
  operator_helm_values          = var.operator_helm_values == "" ? var.operator_helm_values_file == "" ? "" : file(var.operator_helm_values_file) : var.operator_helm_values
}

