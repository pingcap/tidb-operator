resource "google_container_cluster" "cluster" {
  name = var.gke_name
  network = var.vpc_name
  subnetwork = var.subnetwork_name
  location = var.gcp_region
  project = var.gcp_project
}