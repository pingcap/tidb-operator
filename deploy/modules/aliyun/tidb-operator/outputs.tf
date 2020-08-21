output "cluster_id" {
  description = "The id of the ACK cluster."
  value       = alicloud_cs_managed_kubernetes.k8s.id
}

output "cluster_name" {
  description = "The name of ACK cluster"
  value       = alicloud_cs_managed_kubernetes.k8s.name
}

output "vpc_id" {
  description = "The vpc id of ACK cluster"
  value       = var.vpc_id == "" ? alicloud_cs_managed_kubernetes.k8s.vpc_id : var.vpc_id
}

output "vswitch_ids" {
  description = "The available vswich ids of ACK cluster workers"
  value       = split(",", var.vpc_id != "" ? join(",", data.template_file.vswitch_id.*.rendered) : join(",", alicloud_vswitch.all.*.id))
}

output "security_group_id" {
  description = "The security_group_id of ACK cluster"
  value       = var.group_id == "" ? alicloud_cs_managed_kubernetes.k8s.security_group_id : var.group_id
}

output "kubeconfig_filename" {
  description = "The filename of the generated kubectl config."
  value       = data.template_file.kubeconfig_filename.rendered
}

output "region" {
  value = var.region
}

output "bootstrap_token" {
  value = lookup(data.external.token.result, "token")
}

output "key_name" {
  value = var.key_pair_name == "" ? alicloud_key_pair.default[0].key_name : var.key_pair_name
}
