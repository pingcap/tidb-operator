variable "ALICLOUD_ACCESS_KEY" {
}

variable "ALICLOUD_SECRET_KEY" {
}

variable "ALICLOUD_REGION" {
}

provider "alicloud" {
  alias      = "this"
  region     = var.ALICLOUD_REGION
  access_key = var.ALICLOUD_ACCESS_KEY
  secret_key = var.ALICLOUD_SECRET_KEY
}

locals {
  credential_path  = "${path.module}/credentials"
  kubeconfig       = "${local.credential_path}/kubeconfig_${var.ack_name}"
  key_file         = "${local.credential_path}/${var.ack_name}-node-key.pem"
  bastion_key_file = "${local.credential_path}/${var.ack_name}-bastion-key.pem"
}

// AliCloud resource requires path existing
resource "null_resource" "prepare-dir" {
  provisioner "local-exec" {
    command = "mkdir -p ${local.credential_path}"
  }
}

module "tidb-operator" {
  depends_on = [null_resource.prepare-dir]
  source = "./tidb-operator"

  cluster_name         = var.ack_name
  default_worker_count = 2
  vpc_id               = var.vpc_id
  vpc_cidr             = var.vpc_cidr
  vpc_cidr_newbits     = var.vpc_cidr_newbits
  kubeconfig_file      = local.kubeconfig
  group_id             = var.group_id
  operator_version     = var.operator_version
  k8s_pod_cidr         = var.k8s_pod_cidr
  region               = var.ALICLOUD_REGION
  access_key           = var.ALICLOUD_ACCESS_KEY
  access_secret        = var.ALICLOUD_SECRET_KEY
}
