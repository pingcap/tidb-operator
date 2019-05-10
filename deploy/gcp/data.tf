data "google_compute_image" "bastion_image" {
  family = "ubuntu-1804-lts"
  project = "ubuntu-os-cloud"
}