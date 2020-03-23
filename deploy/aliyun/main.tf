variable "ALICLOUD_ACCESS_KEY" {
}

variable "ALICLOUD_SECRET_KEY" {
}

variable "ALICLOUD_REGION" {
}

provider "alicloud" {
  region     = var.ALICLOUD_REGION
  access_key = var.ALICLOUD_ACCESS_KEY
  secret_key = var.ALICLOUD_SECRET_KEY
}

locals {
  credential_path = "${path.cwd}/credentials"
  kubeconfig      = "${local.credential_path}/kubeconfig"
  key_file        = "${local.credential_path}/${var.cluster_name}-key.pem"
}

// AliCloud resource requires path existing
resource "null_resource" "prepare-dir" {
  provisioner "local-exec" {
    command = "mkdir -p ${local.credential_path}"
  }
}

module "tidb-operator" {
  source = "../modules/aliyun/tidb-operator"

  region                        = var.ALICLOUD_REGION
  access_key                    = var.ALICLOUD_ACCESS_KEY
  secret_key                    = var.ALICLOUD_SECRET_KEY
  cluster_name                  = var.cluster_name
  operator_version              = var.operator_version
  operator_helm_values          = var.operator_helm_values == "" ? "" : file(var.operator_helm_values)
  k8s_pod_cidr                  = var.k8s_pod_cidr
  k8s_service_cidr              = var.k8s_service_cidr
  vpc_cidr                      = var.vpc_cidr
  vpc_id                        = var.vpc_id
  default_worker_cpu_core_count = var.default_worker_core_count
  group_id                      = var.group_id
  key_file                      = local.key_file
  kubeconfig_file               = local.kubeconfig
}

module "bastion" {
  source = "../modules/aliyun/bastion"

  bastion_name             = "${var.cluster_name}-bastion"
  key_name                 = module.tidb-operator.key_name
  vpc_id                   = module.tidb-operator.vpc_id
  vswitch_id               = module.tidb-operator.vswitch_ids[0]
  enable_ssh_to_worker     = true
  worker_security_group_id = module.tidb-operator.security_group_id
}

provider "helm" {
  alias          = "default"
  insecure       = true
  install_tiller = false
  kubernetes {
    config_path = module.tidb-operator.kubeconfig_filename
  }
}

module "tidb-cluster" {
  source = "../modules/aliyun/tidb-cluster"
  providers = {
    helm = helm.default
  }

  cluster_name = var.tidb_cluster_name
  ack          = module.tidb-operator

  tidb_version                = var.tidb_version
  tidb_cluster_chart_version  = var.tidb_cluster_chart_version
  pd_instance_type            = var.pd_instance_type
  pd_count                    = var.pd_count
  tikv_instance_type          = var.tikv_instance_type
  tikv_count                  = var.tikv_count
  tidb_instance_type          = var.tidb_instance_type
  tidb_count                  = var.tidb_count
  monitor_instance_type       = var.monitor_instance_type
  create_tidb_cluster_release = var.create_tidb_cluster_release
}
