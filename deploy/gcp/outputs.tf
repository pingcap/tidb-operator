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
