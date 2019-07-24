output "region" {
  value = var.ALICLOUD_REGION
}

output "cluster_id" {
  value = module.tidb-operator.cluster_id
}

output "kubeconfig_file" {
  value = module.tidb-operator.kubeconfig_filename
}

output "vpc_id" {
  value = module.tidb-operator.vpc_id
}

output "bastion_ip" {
  value = module.bastion.bastion_ip
}

output "ssh_key_file" {
  value = local.key_file
}

output "tidb_version" {
  value = var.tidb_version
}

output "tidb_endpoint" {
  value = module.tidb-cluster.tidb_endpoint
}

output "monitor_endpoint" {
  value = module.tidb-cluster.monitor_endpoint
}

