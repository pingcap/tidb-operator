provider "google" {
  credentials = file(var.GCP_CREDENTIALS_PATH)
  region      = var.GCP_REGION
  project     = var.GCP_PROJECT
}

// required for taints on node pools
provider "google-beta" {
  credentials = file(var.GCP_CREDENTIALS_PATH)
  region      = var.GCP_REGION
  project     = var.GCP_PROJECT
}

locals {
  credential_path = "${path.cwd}/credentials"
  kubeconfig      = "${local.credential_path}/kubeconfig_${var.gke_name}"
}


module "project-credentials" {
  source = "../modules/gcp/project-credentials"

  path = local.credential_path
}

module "vpc" {
  source              = "../modules/gcp/vpc"
  create_vpc          = var.create_vpc
  gcp_project         = var.GCP_PROJECT
  gcp_region          = var.GCP_REGION
  vpc_name            = var.vpc_name
  private_subnet_name = "${var.gke_name}-private-subnet"
  public_subnet_name  = "${var.gke_name}-public-subnet"
}

module "tidb-operator" {
  source                        = "../modules/gcp/tidb-operator"
  gke_name                      = var.gke_name
  vpc_name                      = var.vpc_name
  subnetwork_name               = module.vpc.private_subnetwork_name
  gcp_project                   = var.GCP_PROJECT
  gcp_region                    = var.GCP_REGION
  kubeconfig_path               = local.kubeconfig
  tidb_operator_version         = var.tidb_operator_version
  maintenance_window_start_time = var.maintenance_window_start_time
}

module "bastion" {
  source             = "../modules/gcp/bastion"
  vpc_name           = module.vpc.vpc_name
  public_subnet_name = module.vpc.public_subnetwork_name
  gcp_project        = var.GCP_PROJECT
  bastion_name       = "${var.gke_name}-tidb-bastion"
}

module "kubernetes-monitor" {
  source                      = "../modules/gcp/kubernetes-monitor"
  kubeconfig_path             = module.tidb-operator.kubeconfig_path
  install_kubernetes_monitor  = var.install_kubernetes_monitor
  install_prometheus_operator = var.install_prometheus_operator
}
