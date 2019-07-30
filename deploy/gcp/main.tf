variable "GCP_CREDENTIALS_PATH" {
}

variable "GCP_REGION" {
}

variable "GCP_PROJECT" {
}

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
  credential_path          = "${path.cwd}/credentials"
  kubeconfig               = "${local.credential_path}/kubeconfig_${var.gke_name}"
  tidb_cluster_values_path = "${path.module}/rendered/tidb-cluster-values.yaml"
  vpc_name = var.gke_name
}

module "project-credentials" {
  source = "../modules/gcp/project-credentials"

  path = local.credential_path
  gcloud_project = var.GCP_PROJECT
}

module "vpc" {
  source = "../modules/gcp/vpc"
  create_vpc = var.create_vpc
  gcp_project = var.GCP_PROJECT
  gcp_region = var.GCP_REGION
  vpc_name = local.vpc_name
}

//module "tidb-operator" {
//  source = "../modules/gcp/tidb-operator"
//  gke_name = var.gke_name
//  vpc_name = local.vpc_name
//  subnetwork_name = var.private_subnet_name
//  gcp_project = var.GCP_PROJECT
//  gcp_region = var.GCP_REGION
//  pd_count = var.pd_count
//  tikv_count = var.tikv_count
//  tidb_count = var.tidb_count
//  monitor_count = var.monitor_count
//  kubeconfig_path = local.kubeconfig
//  pd_instance_type = var.pd_instance_type
//  tikv_instance_type = var.tikv_instance_type
//  tidb_instance_type = var.tidb_instance_type
//}
//
//module "bastion" {
//  source = "../modules/gcp/bastion"
//  vpc_name = local.vpc_name
//  public_subnet_name = var.public_subnet_name
//  gcp_project = var.GCP_PROJECT
//}
//
//
