resource "google_compute_network" "vpc_network" {
  count = var.create_vpc ? 1 : 0
  name = var.vpc_name
  auto_create_subnetworks = false

}

resource "google_compute_subnetwork" "private_subnet" {
  name = var.private_subnet_name
  ip_cidr_range = var.private_subnet_primary_cidr_range
  network = var.vpc_name
  project = var.gcp_project

  dynamic "secondary_ip_range" {
    for_each = var.private_secondary_ip_ranges_map
    iterator = sir
    content {
      range_name = "${sir.key}-${var.gcp_region}"
      ip_cidr_range = sir.value
    }
  }
  lifecycle {
    ignore_changes = [secondary_ip_range]
  }
}

resource "google_compute_subnetwork" "public_subnet" {
  ip_cidr_range = var.public_subnet_primary_cidr_range
  name = var.public_subnet_name
  network = var.vpc_name
  project = var.gcp_project
}