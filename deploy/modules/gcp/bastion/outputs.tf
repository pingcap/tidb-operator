output "how_to_ssh_to_bastion" {
  value = "gcloud --project ${var.gcp_project} compute ssh ${var.bastion_name} --zone ${google_compute_instance.bastion.zone}"
}
