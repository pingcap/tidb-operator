data "google_compute_zones" "available" {}
data "google_compute_image" "bastion_image" {
  family  = "centos-7"
  project = "centos-cloud"
}
