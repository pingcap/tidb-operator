provider "aws" {
  region = var.region
}

module "key-pair" {
  source = "./aws-key-pair"
  name   = var.eks_name
  path   = "${path.module}/credentials/"
}

module "vpc" {
  source = "terraform-aws-modules/vpc/aws"

  version    = "2.6.0"
  name       = var.eks_name
  cidr       = var.vpc_cidr
  create_vpc = var.create_vpc
  # azs                = [data.aws_availability_zones.available.names[0], data.aws_availability_zones.available.names[1], data.aws_availability_zones.available.names[2]]
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

module "eks" {
  source             = "./eks"
  cluster_name       = var.eks_name
  cluster_version    = var.eks_version
  operator_version   = var.operator_version
  ssh_key_name       = module.key-pair.key_name
  config_output_path = "credentials/"
  subnets            = local.default_subnets
  vpc_id             = var.create_vpc ? module.vpc.vpc_id : var.vpc_id

  tags = {
    app = "tidb"
  }
}
