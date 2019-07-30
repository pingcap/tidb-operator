output "private_subnetwork_name" {
  value = google_compute_subnetwork.private_subnet.name
}

output "public_subnetwork_name" {
  value = google_compute_subnetwork.public_subnet.name
}