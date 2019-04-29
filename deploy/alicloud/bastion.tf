data "alicloud_instance_types" "bastion" {
  count          = "${var.create_bastion ? 1 : 0}"
  cpu_core_count = "${var.bastion_cpu_core_count}"
}

resource "alicloud_key_pair" "bastion" {
  provider        = "alicloud.this"
  count           = "${var.create_bastion ? 1 : 0}"
  key_name_prefix = "${var.bastion_key_prefix}"
  key_file        = "${local.bastion_key_file}"
}

module "bastion-group" {
  source  = "alibaba/security-group/alicloud"
  version = "1.2.0"

  providers = {
    alicloud = "alicloud.this"
  }

  vpc_id            = "${module.ack.vpc_id}"
  cidr_ips          = ["${var.bastion_ingress_cidr}"]
  group_description = "Allow internet SSH connections to bastion node"
  ip_protocols      = ["tcp"]
  port_ranges       = ["22/22"]
  rule_directions   = ["ingress"]
}

resource "alicloud_instance" "bastion" {
  provider                   = "alicloud.this"
  count                      = "${var.create_bastion ? 1 : 0}"
  instance_name              = "${var.cluster_name}-bastion"
  image_id                   = "${var.bastion_image_name}"
  instance_type              = "${data.alicloud_instance_types.bastion.instance_types.0.id}"
  security_groups            = ["${module.bastion-group.security_group_id}"]
  vswitch_id                 = "${module.ack.vswitch_ids[0]}"
  key_name                   = "${alicloud_key_pair.bastion.key_name}"
  internet_charge_type       = "PayByTraffic"
  internet_max_bandwidth_in  = 10
  internet_max_bandwidth_out = 10
  user_data                  = "${file("userdata/bastion-userdata")}"
}
