output "tidb_hostname" {
  value = module.tidb-cluster.tidb_hostname
}

output "monitor_hostname" {
  value = module.tidb-cluster.monitor_hostname
}

output "tidb_endpoint" {
  value = module.tidb-cluster.tidb_endpoint
}

output "monitor_endpoint" {
  value = module.tidb-cluster.monitor_endpoint
}
