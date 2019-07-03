provider "aws" {
  region = var.region
}

locals {
  default_subnets = split(",", var.create_vpc ? join(",", module.vpc.private_subnets) : join(",", var.subnets))
  default_eks     = module.tidb-operator.eks
}

module "key-pair" {
  source = "./aws-key-pair"
  name   = var.eks_name
  path   = "${path.module}/credentials/"
}

module "vpc" {
  source = "terraform-aws-modules/vpc/aws"

  version            = "2.6.0"
  name               = var.eks_name
  cidr               = var.vpc_cidr
  create_vpc         = var.create_vpc
  azs                = data.aws_availability_zones.available.names
  private_subnets    = var.private_subnets
  public_subnets     = var.public_subnets
  enable_nat_gateway = true
  single_nat_gateway = true

  # The following tags are required for ELB
  private_subnet_tags = {
    "kubernetes.io/cluster/${var.eks_name}" = "shared"
  }
  public_subnet_tags = {
    "kubernetes.io/cluster/${var.eks_name}" = "shared"
  }
  vpc_tags = {
    "kubernetes.io/cluster/${var.eks_name}" = "shared"
  }
}

module "tidb-operator" {
  source = "./tidb-operator"

  eks_name           = var.eks_name
  eks_version        = var.eks_version
  operator_version   = var.operator_version
  config_output_path = "credentials/"
  subnets            = local.default_subnets
  vpc_id             = var.create_vpc ? module.vpc.vpc_id : var.vpc_id
  ssh_key_name       = module.key-pair.key_name
}
