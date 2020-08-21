output "kubeconfig_file" {
  value = local.kubeconfig
}

output "how_to_ssh_to_bastion" {
  value = module.bastion.how_to_ssh_to_bastion
}
