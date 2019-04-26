/*
 Alicloud ACK module that launches:

 - A managed kubernetes cluster;
 - Several auto-scaling groups which acting as worker nodes.

 Each auto-scaling group has the same instance type and will
 balance ECS instances across multiple AZ in favor of HA.
 */
provider "alicloud" {
}

resource "alicloud_key_pair" "default" {
  key_name_prefix = "${var.cluster_name}-key"
}

// If there is not specifying vpc_id, create a new one
resource "alicloud_vpc" "vpc" {
  count      = "${var.vpc_id == "" ? 1 : 0}"
  cidr_block = "${var.vpc_cidr}"
  name       = "${var.cluster_name}-vpc"
}

// If no security group id specified, launch one
module "security-group" {
  source   = "alibaba/security-group/alicloud"
  version  = "1.2.0"
  name     = "${var.cluster_name}-group"
  group_id = "${var.security_group_id}"
  vpc_id   = "${var.vpc_id == "" ? join("", alicloud_vpc.vpc.*.id) : var.vpc_id}"
}

// For new VPC, create vswitch for each zone
resource "alicloud_vswitch" "all" {
  count             = "${var.vpc_id == "" ? 0 : length(data.alicloud_zones.all.zones)}"
  vpc_id            = "${alicloud_vpc.vpc.id}"
  cidr_block        = "${cidrsubnet(alicloud_vpc.vpc.cidr_block, var.vpc_cidr_newbits, count.index)}"
  availability_zone = "${lookup(data.alicloud_zones.all.zones[count.index%length(data.alicloud_zones.all.zones)], "id")}"
  name              = "${format("%s-%s", var.cluster_name, format("vsw-%d", count.index+1))}"
}

// Create a managed Kubernetes cluster
resource "alicloud_cs_managed_kuberentes" "this" {
  name        = "${var.cluster_name}"
  vpc_id      = "${var.vpc_id != "" ? var.vpc_id : alicloud_vpc.vpc.id}"
  vswitch_ids = "${var.vpc_id != "" ? data.alicloud_vswitches.default.*.id : alicloud_vswitch.all.*.id}"

  // Default auto-scaling group created by managed kubernetes cannot span across AZ,
  // so we don't provision any worker in default group
  worker_numbers = [0]

  worker_instance_type = "ecs.t5.xlarge"                         // placeholder
  key_name             = "${alicloud_key_pair.default.key_name}"
  pod_cidr             = "${var.k8s_pod_cidr}"
  service_cidr         = "${var.k8s_service_cidr}"
  new_nat_gateway      = "${var.create_nat_gateway}"
  cluster_network_type = "${var.cluster_network_type}"
  slb_internet_enabled = "${var.public_apiserver}"
}

// Create auto-scaling groups
resource "alicloud_ess_scaling_groups" "workers" {
  count              = "${length(var.worker_groups)}"
  scaling_group_name = "${alicloud_cs_managed_kuberentes.this.name}-${lookup(var.worker_groups[count.index], "name", count.index)}"
  vswitch_ids        = "${var.vpc_id != "" ? data.alicloud_vswitches.default.*.id : alicloud_vswitch.all.*.id}"
  min_size           = "${lookup(var.worker_groups[count.index], "min_size", var.group_default["min_size"])}"
  max_size           = "${lookup(var.worker_groups[count.index], "max_size", var.group_default["max_size"])}"
  default_cooldown   = "${lookup(var.worker_groups[count.index], "default_cooldown", var.group_default["default_cooldown"])}"
  multi_az_policy    = "${lookup(var.worker_groups[count.index], "multi_az_policy", var.group_default["multi_az_policy"])}"
}

// Create the cooresponding auto-scaling configurations
resource "alicloud_ess_scaling_configuration" "workers" {
  count                = "${length(var.worker_groups)}"
  scaling_group_id     = "${alicloud_ess_scaling_groups.workers[count.index].id}"
  image_id             = "${lookup(var.worker_groups[count.index], "image_id", var.group_default["image_id"])}"
  instance_type        = "${lookup(var.worker_groups[count.index], "instance_type", var.group_default["instance_type"])}"
  security_group_id    = "${var.security_group_id != "" ? var.security_group_id : module.security-group.security_group_id}"
  key_name             = "${alicloud_key_pair.default.key_name}"
  system_disk_category = "${lookup(var.worker_groups[count.index], "system_disk_category", var.group_default["system_disk_category"])}"
  system_disk_size     = "${lookup(var.worker_groups[count.index], "system_disk_size", var.group_default["system_disk_size"])}"
  data_disk            = "${lookup(var.worker_groups[count.index], "data_disk", var.group_default["data_disk"])}"
  user_data            = "${element(data.template_file.userdata.*.rendered, count.index)}"
  enable               = true
  active               = true
//
//  tags = "${merge(map(
//        "name", "${alicloud_cs_managed_kubernetes.this.name}-${lookup(var.worker_groups[count.index], "name", count.index)}-ack_asg",
//        "kubernetes.io/cluster/${alicloud_cs_managed_kubernetes.this.name}", "owned",
//        "k8s.io/cluster-autoscaler/${lookup(var.worker_groups[count.index], "autoscaling_enabled", var.group_default["autoscaling_enabled"]) == 1 ? "enabled" : "disabled"}", "true",
//        "k8s.io/cluster-autoscaler/${alicloud_cs_managed_kubernetes.this.name}", ""
//      ),
//      var.group_tags,
//      var.worker_groups[count.index].tags
//    )
//  }"
}
