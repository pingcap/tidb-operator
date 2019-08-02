output "region" {
  value = var.GCP_REGION
}

output "kubeconfig_file" {
  value = local.kubeconfig
}

output "tidb_version" {
  value = var.tidb_version
}

output "monitor_ilb_ip" {
  value = module.default-tidb-cluster.monitor_ilb_ip
}

output "monitor_port" {
  value = module.default-tidb-cluster.monitor_port
}

output "how_to_ssh_to_bastion" {
  value = module.bastion.how_to_ssh_to_bastion
}

output "how_to_connect_to_default_cluster_tidb_from_bastion" {
  value = module.default-tidb-cluster.how_to_connect_to_mysql_from_bastion
}

