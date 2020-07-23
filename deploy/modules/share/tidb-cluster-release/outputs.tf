locals {
  tidb_hostname    = var.create ? lookup(data.external.tidb_hostname[0].result, var.service_ingress_key, "empty") : "not_created"
  monitor_hostname = var.create ? lookup(data.external.monitor_hostname[0].result, var.service_ingress_key, "empty") : "not_created"
}

output "tidb_hostname" {
  value = local.tidb_hostname
}

output "monitor_hostname" {
  value = local.monitor_hostname
}

output "tidb_endpoint" {
  value = var.create ? "${local.tidb_hostname}:${data.external.tidb_port[0].result["port"]}" : "not_created"
}

output "monitor_endpoint" {
  value = var.create ? "${local.monitor_hostname}:${data.external.monitor_port[0].result["port"]}" : "not_created"
}
