output "region" {
  value = var.ALICLOUD_REGION
}

output "kubeconfig_file" {
  value = module.tidb-operator.kubeconfig_filename
}

output "vpc_id" {
  value = module.tidb-operator.vpc_id
}

output "bastion_ip" {
  value = join(",", alicloud_instance.bastion.*.public_ip)
}

output "bastion_key_file" {
  value = local.bastion_key_file
}

output "worker_key_file" {
  value = local.key_file
}

output "default_cluster_tidb_slb_ip" {
  value = module.default-cluster.tidb_slb_ip
}

output "default_cluster_tidb_port" {
  value = module.default-cluster.tidb_port
}

output "default_cluster_monitor_endpoint" {
  value = module.default-cluster.monitor_endpoint
}

