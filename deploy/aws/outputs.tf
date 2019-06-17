output "region" {
  description = "AWS region."
  value       = var.region
}

output "eks_version" {
  description = "The Kubernetes server version for the EKS cluster."
  value       = var.eks_version
}

output "eks_endpoint" {
  description = "Endpoint for EKS control plane."
  value       = module.eks.cluster_endpoint
}

output "demo-cluster_tidb-dns" {
  description = "tidb service endpoints"
  value       = module.demo-cluster.tidb_dns
}

output "demo-cluster_monitor-dns" {
  description = "tidb service endpoint"
  value       = module.demo-cluster.monitor_dns
}

output "test-cluster_tidb-dns" {
  description = "tidb service endpoints"
  value       = module.test-cluster.tidb_dns
}

output "test-cluster_monitor-dns" {
  description = "tidb service endpoint"
  value       = module.test-cluster.monitor_dns
}

output "prod-cluster_tidb-dns" {
  description = "tidb service endpoints"
  value       = module.prod-cluster.tidb_dns
}

output "prod-cluster_monitor-dns" {
  description = "tidb service endpoint"
  value       = module.prod-cluster.monitor_dns
}

# output "bastion_ip" {
#   description = "Bastion IP address"
#   value       = module.ec2.public_ip
# }
