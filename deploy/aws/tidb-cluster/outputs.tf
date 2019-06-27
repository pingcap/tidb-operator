output "tidb_dns" {
  value = data.external.tidb_elb.result["hostname"]
}

output "monitor_dns" {
  value = data.external.monitor_elb.result["hostname"]
}
