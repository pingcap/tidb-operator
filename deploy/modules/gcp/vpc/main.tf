resource "google_compute_network" "vpc_network" {
  count                   = var.create_vpc ? 1 : 0
  name                    = var.vpc_name
  auto_create_subnetworks = false

}

resource "google_compute_subnetwork" "private_subnet" {
  depends_on    = [google_compute_network.vpc_network]
  name          = var.private_subnet_name
  ip_cidr_range = var.private_subnet_primary_cidr_range
  network       = var.vpc_name
  project       = var.gcp_project

  secondary_ip_range {
    ip_cidr_range = var.private_subnet_secondary_cidr_ranges[0]
    range_name    = "pods-${var.gcp_region}"
  }

  secondary_ip_range {
    ip_cidr_range = var.private_subnet_secondary_cidr_ranges[1]
    range_name    = "services-${var.gcp_region}"
  }


  lifecycle {
    ignore_changes = [secondary_ip_range]
  }
}

resource "google_compute_subnetwork" "public_subnet" {
  depends_on    = [google_compute_network.vpc_network]
  ip_cidr_range = var.public_subnet_primary_cidr_range
  name          = var.public_subnet_name
  network       = var.vpc_name
  project       = var.gcp_project
}
