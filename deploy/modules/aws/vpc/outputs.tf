output "vpc_id" {
  value = var.create_vpc ? module.vpc.vpc_id : var.vpc_id
}

output "public_subnets" {
  value = var.create_vpc ? module.vpc.public_subnets : var.public_subnets
}

output "private_subnets" {
  value = var.create_vpc ? module.vpc.private_subnets : var.private_subnets
}
