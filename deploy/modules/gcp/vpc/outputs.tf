output "vpc_name" {
  value = var.create_vpc ? google_compute_network.vpc_network[0].name : var.vpc_name
}

output "private_subnetwork_name" {
  value = google_compute_subnetwork.private_subnet.name
}

output "public_subnetwork_name" {
  value = google_compute_subnetwork.public_subnet.name
}
