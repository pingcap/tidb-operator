output "bastion_ip" {
  value = join(",", alicloud_instance.bastion.*.public_ip)
}
