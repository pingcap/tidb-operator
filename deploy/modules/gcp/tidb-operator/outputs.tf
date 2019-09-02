output "cluster_id" {
  value = google_container_cluster.cluster.id
}

output "tidb_operator_id" {
  value = helm_release.tidb-operator.id
}

output "get_credentials_id" {
  value = null_resource.get-credentials.id
}
