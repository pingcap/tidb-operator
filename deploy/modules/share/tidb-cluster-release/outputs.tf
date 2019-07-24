locals {
  tidb_hostname = lookup(data.external.tidb_hostname.result, var.service_ingress_key, "empty")
  monitor_hostname = lookup(data.external.monitor_hostname.result, var.service_ingress_key, "emtpy")
}

output "tidb_hostname" {
  value = local.tidb_hostname
}

output "monitor_hostname" {
  value = local.monitor_hostname
}

output "tidb_endpoint" {
  value = "${local.tidb_hostname}:${data.external.tidb_port.result["port"]}"
}

output "monitor_endpoint" {
  value = "${local.monitor_hostname}:${data.external.monitor_port.result["port"]}"
}
