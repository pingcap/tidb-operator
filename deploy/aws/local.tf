locals {
  default_subnets = split(",", var.create_vpc ? join(",", module.vpc.private_subnets) : join(",", var.subnets))
  default_eks     = module.eks.eks_info
  kubeconfig      = "${path.module}/credentials/kubeconfig-${var.eks_name}"
}