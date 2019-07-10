output "cluster_id" {
  description = "The id of the ACK cluster."
  value       = alicloud_cs_managed_kubernetes.k8s.*.id
}

output "cluster_name" {
  description = "The name of ACK cluster"
  value       = alicloud_cs_managed_kubernetes.k8s.*.name
}

output "vpc_id" {
  description = "The vpc id of ACK cluster"
  value       = alicloud_cs_managed_kubernetes.k8s.*.vpc_id
}

output "vswitch_ids" {
  description = "The vswich ids of ACK cluster"
  value       = alicloud_cs_managed_kubernetes.k8s.*.vswitch_ids
}

output "security_group_id" {
  description = "The security_group_id of ACK cluster"
  value       = alicloud_cs_managed_kubernetes.k8s.*.security_group_id
}

output "key_name" {
  description = "The key pair for ACK worker nodes"
  value       = var.key_pair_name == "" ? alicloud_key_pair.default.key_name : var.key_pair_name
}

output "kubeconfig" {
  description = "The kubeconfig file content"
  value       = file(var.kubeconfig_file)
}

output "kubeconfig_filename" {
  description = "The filename of the generated kubectl config."
  value       = var.kubeconfig_file
}

output "region" {
  value = var.region
}

output "bootstrap_token" {
  value = data.external.token
}
