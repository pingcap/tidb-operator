/*
 Alicloud ACK module that launches:

 - A managed kubernetes cluster;
 - Several auto-scaling groups which acting as worker nodes.

 Each auto-scaling group has the same instance type and will
 balance ECS instances across multiple AZ in favor of HA.
 */
provider "alicloud" {}

resource "alicloud_key_pair" "default" {
  count           = "${var.key_pair_name == "" ? 1 : 0}"
  key_name_prefix = "${var.cluster_name_prefix}-key"
  key_file        = "${var.key_file != "" ? var.key_file : format("%s/%s-key", path.module, var.cluster_name_prefix)}"
}

# If there is not specifying vpc_id, create a new one
resource "alicloud_vpc" "vpc" {
  count      = "${var.vpc_id == "" ? 1 : 0}"
  cidr_block = "${var.vpc_cidr}"
  name       = "${var.cluster_name_prefix}-vpc"

  lifecycle {
    ignore_changes = ["cidr_block"]
  }
}

# For new vpc or existing vpc with no vswitches, create vswitch for each zone
resource "alicloud_vswitch" "all" {
  count             = "${var.vpc_id != "" && (length(data.alicloud_vswitches.default.vswitches) != 0) ? 0 : length(data.alicloud_zones.all.zones)}"
  vpc_id            = "${alicloud_vpc.vpc.0.id}"
  cidr_block        = "${cidrsubnet(alicloud_vpc.vpc.0.cidr_block, var.vpc_cidr_newbits, count.index)}"
  availability_zone = "${lookup(data.alicloud_zones.all.zones[count.index%length(data.alicloud_zones.all.zones)], "id")}"
  name              = "${format("vsw-%s-%d", var.cluster_name_prefix, count.index+1)}"
}

resource "alicloud_security_group" "group" {
  count       = "${var.group_id == "" ? 1 : 0}"
  name        = "${var.cluster_name_prefix}-sg"
  vpc_id      = "${var.vpc_id != "" ? var.vpc_id : alicloud_vpc.vpc.0.id}"
  description = "Security group for ACK worker nodes"
}

# Allow traffic inside VPC
resource "alicloud_security_group_rule" "cluster_worker_ingress" {
  count             = "${var.group_id == "" ? 1 : 0}"
  security_group_id = "${alicloud_security_group.group.id}"
  type              = "ingress"
  ip_protocol       = "all"
  nic_type          = "intranet"
  port_range        = "-1/-1"
  cidr_ip           = "${var.vpc_id != "" ? var.vpc_cidr : alicloud_vpc.vpc.0.cidr_block}"
}

# Create a managed Kubernetes cluster
resource "alicloud_cs_managed_kubernetes" "k8s" {
  name_prefix = "${var.cluster_name_prefix}"

  // split and join: workaround for terraform's limitation of conditional list choice, similarly hereinafter
  vswitch_ids           = ["${element(split(",", var.vpc_id != "" && (length(data.alicloud_vswitches.default.vswitches) != 0) ? join(",", data.template_file.vswitch_id.*.rendered) : join(",", alicloud_vswitch.all.*.id)), 0)}"]
  key_name              = "${alicloud_key_pair.default.key_name}"
  pod_cidr              = "${var.k8s_pod_cidr}"
  service_cidr          = "${var.k8s_service_cidr}"
  new_nat_gateway       = "${var.create_nat_gateway}"
  cluster_network_type  = "${var.cluster_network_type}"
  slb_internet_enabled  = "${var.public_apiserver}"
  kube_config           = "${var.kubeconfig_file != "" ? var.kubeconfig_file : format("%s/kubeconfig", path.module)}"
  worker_numbers        = ["${var.default_worker_count}"]
  worker_instance_types = ["${var.default_worker_type != "" ? var.default_worker_type : data.alicloud_instance_types.default.instance_types.0.id}"]

  # These varialbes are 'ForceNew' that will cause kubernetes cluster re-creation
  # on variable change, so we make all these variables immutable in favor of safety.
  lifecycle {
    ignore_changes = [
      "vswitch_ids",
      "worker_instance_types",
      "key_name",
      "pod_cidr",
      "service_cidr",
      "cluster_network_type",
    ]
  }

  depends_on = ["alicloud_vpc.vpc"]
}

# Create auto-scaling groups
resource "alicloud_ess_scaling_group" "workers" {
  count              = "${length(var.worker_groups)}"
  scaling_group_name = "${alicloud_cs_managed_kubernetes.k8s.name}-${lookup(var.worker_groups[count.index], "name", count.index)}"
  vswitch_ids        = ["${split(",", var.vpc_id != "" ? join(",", data.template_file.vswitch_id.*.rendered) : join(",", alicloud_vswitch.all.*.id))}"]
  min_size           = "${lookup(var.worker_groups[count.index], "min_size", var.group_default["min_size"])}"
  max_size           = "${lookup(var.worker_groups[count.index], "max_size", var.group_default["max_size"])}"
  default_cooldown   = "${lookup(var.worker_groups[count.index], "default_cooldown", var.group_default["default_cooldown"])}"
  multi_az_policy    = "${lookup(var.worker_groups[count.index], "multi_az_policy", var.group_default["multi_az_policy"])}"

  # Remove the newest instance in the oldest scaling configuration
  removal_policies = [
    "OldestScalingConfiguration",
    "NewestInstance",
  ]

  lifecycle {
    # FIXME: currently update vswitch_ids will force will recreate, allow updating when upstream support in-place
    # vswitch id update
    ignore_changes = ["vswitch_ids"]

    create_before_destroy = true
  }
}

# Create the cooresponding auto-scaling configurations
resource "alicloud_ess_scaling_configuration" "workers" {
  count                      = "${length(var.worker_groups)}"
  scaling_group_id           = "${element(alicloud_ess_scaling_group.workers.*.id, count.index)}"
  image_id                   = "${lookup(var.worker_groups[count.index], "image_id", var.group_default["image_id"])}"
  instance_type              = "${lookup(var.worker_groups[count.index], "instance_type", var.group_default["instance_type"])}"
  security_group_id          = "${var.group_id != "" ? var.group_id : alicloud_security_group.group.id}"
  key_name                   = "${alicloud_key_pair.default.key_name}"
  system_disk_category       = "${lookup(var.worker_groups[count.index], "system_disk_category", var.group_default["system_disk_category"])}"
  system_disk_size           = "${lookup(var.worker_groups[count.index], "system_disk_size", var.group_default["system_disk_size"])}"
  user_data                  = "${element(data.template_file.userdata.*.rendered, count.index)}"
  internet_charge_type       = "${lookup(var.worker_groups[count.index], "internet_charge_type", var.group_default["internet_charge_type"])}"
  internet_max_bandwidth_in  = "${lookup(var.worker_groups[count.index], "internet_max_bandwidth_in", var.group_default["internet_max_bandwidth_in"])}"
  internet_max_bandwidth_out = "${lookup(var.worker_groups[count.index], "internet_max_bandwidth_out", var.group_default["internet_max_bandwidth_out"])}"

  enable       = true
  active       = true
  force_delete = true

  tags = "${merge(map(
          "name", "${alicloud_cs_managed_kubernetes.k8s.name}-${lookup(var.worker_groups[count.index], "name", count.index)}-ack_asg",
          "kubernetes.io/cluster/${alicloud_cs_managed_kubernetes.k8s.name}", "owned",
          "k8s.io/cluster-autoscaler/${lookup(var.worker_groups[count.index], "autoscaling_enabled", var.group_default["autoscaling_enabled"]) == 1 ? "enabled" : "disabled"}", "true",
          "k8s.io/cluster-autoscaler/${alicloud_cs_managed_kubernetes.k8s.name}", "default"
        ),
        var.default_group_tags,
        var.worker_group_tags[count.index%length(var.worker_group_tags)]
      )
    }"

  lifecycle {
    ignore_changes        = ["instance_type"]
    create_before_destroy = true
  }
}
