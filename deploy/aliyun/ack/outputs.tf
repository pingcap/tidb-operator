output "cluster_id" {
  description = "The id of the ACK cluster."
  value       = "${alicloud_cs_managed_kubernetes.k8s.*.id}"
}

output "cluster_name" {
  description = "The name of ACK cluster"
  value       = "${alicloud_cs_managed_kubernetes.k8s.*.id}"
}

output "cluster_nodes" {
  description = "The cluster worker node ids of ACK cluster"
  value       = "${alicloud_ess_scaling_configuration.workers.*.id}"
}

output "vpc_id" {
  description = "The vpc id of ACK cluster"
  value       = "${alicloud_cs_managed_kubernetes.k8s.*.vpc_id}"
}

output "vswitch_ids" {
  description = "The vswich ids of ACK cluster"
  value       = "${alicloud_cs_managed_kubernetes.k8s.*.vswitch_ids}"
}

output "security_group_id" {
  description = "The security_group_id of ACK cluster"
  value       = "${alicloud_cs_managed_kubernetes.k8s.*.security_group_id}"
}

output "kubeconfig_filename" {
  description = "The filename of the generated kubectl config."
  value       = "${path.module}/kubeconfig_${var.cluster_name_prefix}"
}
