output "tidb_version" {
  value = var.cluster_version
}

output "tidb_ilb_ip" {
  value = data.external.tidb_ilb_ip.result["ip"]
}

output "tidb_port" {
  value = data.external.tidb_port.result["port"]
}

output "monitor_lb_ip" {
  value = data.external.monitor_lb_ip.result["ip"]
}

output "monitor_port" {
  value = data.external.monitor_port.result["port"]
}


output "how_to_connect_to_tidb_from_bastion" {
  value = "mysql -h ${data.external.tidb_ilb_ip.result["ip"]} -P ${data.external.tidb_port.result["port"]} -u root"
}

output "tidb_pool" {
  value = google_container_node_pool.tidb_pool
}

output "how_to_set_reclaim_policy_to_delete" {
  description = "The kubectl command for changing the ReclaimPolicy for persistent volumes claimed by a TiDB cluster to Delete to avoid orphaned disks. Run this command before terraform destroy."
  value       = "kubectl --kubeconfig ${var.kubeconfig_path} get pvc -n ${var.cluster_name} -o jsonpath='{.items[*].spec.volumeName}'|fmt -1 | xargs -I {} kubectl --kubeconfig ${var.kubeconfig_path} patch pv {} -p '{\"spec\":{\"persistentVolumeReclaimPolicy\":\"Delete\"}}'"
}
