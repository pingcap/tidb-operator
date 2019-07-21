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
  value       = module.tidb-operator.eks.cluster_endpoint
}

output "kubeconfig_filename" {
  description = "The filename of the generated kubectl config."
  value       = module.tidb-operator.eks.kubeconfig_filename
}

output "default-cluster_tidb-dns" {
  description = "tidb service endpoints"
  value       = module.default-cluster.tidb_hostname
}

output "default-cluster_monitor-dns" {
  description = "tidb service endpoint"
  value       = module.default-cluster.monitor_hostname
}

output "bastion_ip" {
  description = "Bastion IP address"
  value       = module.bastion.bastion_ip
}
