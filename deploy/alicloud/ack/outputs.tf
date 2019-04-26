output "cluster_id" {
  description = "The id of the ACK cluster."
  value       = "${alicloud_cs_managed_kuberentes.this.id}"
}

output "cluster_name" {
  description = "The name of ACK cluster"
  value       = "${alicloud_cs_managed_kuberentes.this.name}"
}

output "cluster_nodes" {
  description = "The cluster worker nodes of ACK cluster"
  value       = "${alicloud_ess_scaling_configuration.workers}"
}

output "vpc_id" {
  description = "The vpc id of ACK cluster"
  value       = "${alicloud_cs_managed_kuberentes.this.vpc_id}"
}

output "nat_gateway_id" {
  description = "The nat gateway id of ACK cluster"
  value       = "${alicloud_cs_managed_kuberentes.this.nat_gateway_id}"
}

output "security_group_id" {
  description = "The security_group_id of ACK cluster"
  value       = "${alicloud_cs_managed_kuberentes.this.security_group_id}"
}

output "kubeconfig_filename" {
  description = "The filename of the generated kubectl config."
  value       = "${path.module}/kubeconfig_${var.cluster_name}"
}
