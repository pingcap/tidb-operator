data "alicloud_instance_types" "bastion" {
  cpu_core_count = var.bastion_cpu_core_count
}

resource "alicloud_security_group" "bastion-group" {
  name        = var.bastion_name
  vpc_id      = var.vpc_id
  description = "Allow internet SSH connections to bastion node"
}

resource "alicloud_security_group_rule" "allow_ssh_from_local" {
  type              = "ingress"
  ip_protocol       = "tcp"
  nic_type          = "intranet"
  port_range        = "22/22"
  security_group_id = alicloud_security_group.bastion-group.id
  cidr_ip           = var.bastion_ingress_cidr
}

resource "alicloud_security_group_rule" "allow_ssh_to_worker" {
  count                    = var.enable_ssh_to_worker ? 1 : 0
  type                     = "ingress"
  ip_protocol              = "tcp"
  nic_type                 = "intranet"
  policy                   = "accept"
  port_range               = "22/22"
  priority                 = 1
  security_group_id        = var.worker_security_group_id
  source_security_group_id = alicloud_security_group.bastion-group.id
}

resource "alicloud_instance" "bastion" {
  instance_name              = var.bastion_name
  image_id                   = var.bastion_image_name
  instance_type              = data.alicloud_instance_types.bastion.instance_types[0].id
  security_groups            = [alicloud_security_group.bastion-group.id]
  vswitch_id                 = var.vswitch_id
  key_name                   = var.key_name
  internet_charge_type       = "PayByTraffic"
  internet_max_bandwidth_in  = 10
  internet_max_bandwidth_out = 10
  user_data                  = file("${path.module}/bastion-userdata")
}
