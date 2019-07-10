output "tidb_slb_ip" {
  value = data.external.tidb_slb_ip.result["ip"]
}

output "tidb_port" {
  value = data.external.tidb_port.result["port"]
}

output "monitor_endpoint" {
  value = "${data.external.monitor_slb_ip.result["ip"]}:${data.external.monitor_port.result["port"]}"
}