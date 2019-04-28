output "cluster_id" {
  value = "${module.ack.cluster_id}"
}

output "kubeconfig_file" {
  value = "${module.ack.kubeconfig_filename}"
}

output "vpc_id" {
  value = "${module.ack.vpc_id}"
}

output "bastion_ip" {
  value = "${join(",", alicloud_instance.bastion.*.public_ip)}"
}

output "bastion_key_file" {
  value = "${local.bastion_key_file}"
}

output "worker_key_file" {
  value = "${local.key_file}"
}
