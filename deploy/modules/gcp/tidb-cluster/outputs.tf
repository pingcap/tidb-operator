output "tidb_version" {
  value = var.cluster_version
}

output "tidb_ilb_ip" {
  value = data.external.tidb_ilb_ip.result["ip"]
}

output "tidb_port" {
  value = data.external.tidb_port.result["port"]
}

output "monitor_ilb_ip" {
  value = data.external.monitor_ilb_ip.result["ip"]
}

output "monitor_port" {
  value = data.external.monitor_port.result["port"]
}


output "how_to_connect_to_mysql_from_bastion" {
  value = "mysql -h ${data.external.tidb_ilb_ip.result["ip"]} -P ${data.external.tidb_port.result["port"]} -u root"
}