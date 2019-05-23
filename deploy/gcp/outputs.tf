output "region" {
  value = "${var.GCP_REGION}"
}

output "cluster_id" {
  value = "${google_container_cluster.cluster.id}"
}

output "cluster_name" {
  value = "${google_container_cluster.cluster.name}"
}

output "kubeconfig_file" {
  value = "${local.kubeconfig}"
}

output "tidb_version" {
  value = "${var.tidb_version}"
}

output "tidb_ilb_ip" {
  value = "${data.external.tidb_ilb_ip.result["ip"]}"
}

output "tidb_port" {
  value = "${data.external.tidb_port.result["port"]}"
}

output "monitor_ilb_ip" {
  value = "${data.external.monitor_ilb_ip.result["ip"]}"
}

output "monitor_port" {
  value = "${data.external.monitor_port.result["port"]}"
}

output "how_to_ssh_to_bastion" {
  value = "gcloud compute ssh bastion --zone ${var.GCP_REGION}-a"
}

output "how_to_connect_to_mysql_from_bastion" {
  value = "mysql -h ${data.external.tidb_ilb_ip.result["ip"]} -P ${data.external.tidb_port.result["port"]} -u root"
}
