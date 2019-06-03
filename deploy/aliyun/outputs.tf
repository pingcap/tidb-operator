output "region" {
  value = "${var.ALICLOUD_REGION}"
}

output "cluster_id" {
  value = "${module.ack.cluster_id}"
}

output "cluster_name" {
  value = "${var.cluster_name_prefix}"
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

output "tidb_version" {
  value = "${var.tidb_version}"
}

output "tidb_slb_ip" {
  value = "${data.external.tidb_slb_ip.result["ip"]}"
}

output "tidb_port" {
  value = "${data.external.tidb_port.result["port"]}"
}

output "monitor_endpoint" {
  value = "${data.external.monitor_slb_ip.result["ip"]}:${data.external.monitor_port.result["port"]}"
}
