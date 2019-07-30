output "how_to_ssh_to_bastion" {
  value = "gcloud compute ssh bastion --zone ${google_compute_instance.bastion.zone}"
}
