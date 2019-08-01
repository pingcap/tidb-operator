output "cluster_id" {
  value = google_container_cluster.cluster.id
}

output "gke_cluster_name" {
  value = google_container_cluster.cluster.name
}

output "gcp_project" {
  value = google_container_cluster.cluster.project
}

output "gke_cluster_location" {
  value = google_container_cluster.cluster.location
}

output "kubeconfig_path" {
  value = var.kubeconfig_path
}

output "tidb_operator_version" {
  value = helm_release.tidb-operator.version
}