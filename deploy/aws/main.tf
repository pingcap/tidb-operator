provider "aws" {
  region = var.region
}

locals {
  eks     = module.tidb-operator.eks
  subnets = module.vpc.private_subnets
}

module "key-pair" {
  source = "../modules/aws/key-pair"

  name = var.eks_name
  path = "${path.cwd}/credentials/"
}

module "vpc" {
  source = "../modules/aws/vpc"

  vpc_name        = var.eks_name
  create_vpc      = var.create_vpc
  private_subnets = var.private_subnets
  public_subnets  = var.public_subnets
  vpc_cidr        = var.vpc_cidr
}

module "tidb-operator" {
  source = "../modules/aws/tidb-operator"

  eks_name           = var.eks_name
  eks_version        = var.eks_version
  operator_version   = var.operator_version
  config_output_path = "credentials/"
  subnets            = local.subnets
  vpc_id             = module.vpc.vpc_id
  ssh_key_name       = module.key-pair.key_name
}

module "bastion" {
  source = "../modules/aws/bastion"

  bastion_name             = "${var.eks_name}-bastion"
  key_name                 = module.key-pair.key_name
  public_subnets           = module.vpc.public_subnets
  vpc_id                   = module.vpc.vpc_id
  worker_security_group_id = local.eks.worker_security_group_id
  enable_ssh_to_workers    = true
}
