output "tidb_dns" {
  value = lookup(data.external.tidb_elb.result, "hostname", "empty")
}

output "monitor_dns" {
  value = lookup(data.external.monitor_elb.result, "hostname", "emtpy")
}
