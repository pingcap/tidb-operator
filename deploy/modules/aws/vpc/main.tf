provider "aws" {
  region = var.region
}

data "aws_availability_zones" "available" {
}

module "vpc" {
  source = "terraform-aws-modules/vpc/aws"

  version            = "2.6.0"
  name               = var.vpc_name
  cidr               = var.vpc_cidr
  create_vpc         = var.create_vpc
  azs                = data.aws_availability_zones.available.names
  private_subnets    = var.private_subnets
  public_subnets     = var.public_subnets
  enable_nat_gateway = true
  single_nat_gateway = true

  # The following tags are required for ELB
  private_subnet_tags = {
    "kubernetes.io/cluster/${var.vpc_name}" = "shared"
  }
  public_subnet_tags = {
    "kubernetes.io/cluster/${var.vpc_name}" = "shared"
  }
  vpc_tags = {
    "kubernetes.io/cluster/${var.vpc_name}" = "shared"
  }
}
