output "tidb_hostname" {
  value = lookup(data.external.tidb_hostname.result, "hostname", "empty")
}

output "monitor_hostname" {
  value = lookup(data.external.monitor_hostname.result, "hostname", "emtpy")
}
