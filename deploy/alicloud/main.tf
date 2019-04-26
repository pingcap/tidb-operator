provider "alicloud" {
  access_key = "${var.alicloud_access_key}"
  secret_key = "${var.alicloud.secret.key}"
  region = "${var.region}"
}

data "alicloud_instance_types" "pd" {
  instance_type_family = "${var.pd_type_family}"
  cpu_core_count = "${var.pd_cpu}"
  network_type = "Vpc"
}

data "alicloud_instance_types" "tikv" {
  instance_type_family = "${var.tikv_type_family}"
  cpu_core_count = "${var.tikv_cpu}"
  network_type = "Vpc"
}

data "alicloud_instance_types" "tidb" {
  instance_type_family = "${var.tidb_type_family}"
  cpu_core_count = "${var.tidb_cpu}"
  network_type = "Vpc"
}

// Choose AZ accroding to instance types needed
data "alicloud_zones" "default" {
  available_instance_type = "${setintersection(data.alicloud_instance_types.*.instance_types).0.id}"
}

resource "alicloud_key_pair" "default" {
  key_name_prefix = "${var.cluster_name}"
}

// If there is not specifying vpc_id, launch a new vpc
resource "alicloud_vpc" "vpc" {
  count = "${var.vpc_id == "" ? 1 : 0}"
  cidr_block = "${var.vpc_cidr}"
  name = "${var.cluster_name}"
}

// Launch several vswitches according to the vswitch cidr blocks
resource "alicloud_vswitch" "default" {
  count = "${var.vswitch_id == "" 1 : 0}"
  vpc_id = "${var.vpc_id == "" ? join("", alicloud_vpc.vpc.*.id) : var.vpc_id}"
  cidr_block = "${var.vswitch_cidr}"
  availability_zone = "${data.alicloud_zones.default.zones.0.id}"
  name = "${va.cluster_name}"
}

// If apiserver public accessing is needed, launch a NAT Gateway associating with an EIP
resource "alicloud_nat_gateway" "default" {
  count = "${var.create_eip == true ? 1 : 0}"
  vpc_id = "${var.vpc_id == "" ? join("", alicloud_vpc.vpc.*.id) : var.vpc_id}"
  name = "${var.cluster_name}"
}

resource "alicloud_eip" "default" {
  count = "${var.create_eip == "true" ? 1 : 0}"
  bandwidth = 10
}

resource "alicloud_eip_association" "default" {
  count = "${var.create_eip == "true" ? 1 : 0}"
  allocation_id = "${alicloud_eip.default.id}"
  instance_id = "${alicloud_nat_gateway.default.id}"
}

resource "alicloud_snat_entry" "default"{
  count = "${var.create_eip == "false" ? 0 : 1}"
  snat_table_id = "${alicloud_nat_gateway.default.snat_table_ids}"
  source_vswitch_id = "${var.vswich_id != "" ? var.vswitch_id : alicloud_vswitch.default.id}"
  snat_ip = "${alicloud_eip.default.ip_address}"
}

// Launch a managed Kubernetes cluster
resource "alicloud_cs_managed_kuberentes" "defaut" {
  name = "${var.cluster_name}"
  vswitch_id = "${var.vswich_id != "" ? var.vswitch_id : alicloud_vswitch.default.id}"
  new_nat_gateway = false
  // set default worker instance type according to tidb node type in favor of scaling
  worker_instance_type = "${data.alicloud_instance_types.tidb.instance_types.0.id}"
  worker_number = "${var.tidb_count}"
  worker_disk_category = "${var.worker_disk_category}"
  worker_disk_size = "${var.worker_disk_size}"
  key_name = "${alicloud_key_pair.default.key_name}"
  pod_cidr = "${var.k8s_pod_cidr}"
  service_cidr = "${var.k8s_service_cidr}"
  enable_ssh = ${var.enable_ssh}
  install_cloud_monitor = ${var.install_cloud_monitor}
}

// If no security group id specified, launch one
module "security-group" {
  source  = "alibaba/security-group/alicloud"
  version = "1.2.0"
  group_id = "${var.security_group_id}"
  vpc_id = "${var.vpc_id == "" ? join("", alicloud_vpc.vpc.*.id) : var.vpc_id}"
  name = "${var.cluster_name}"
}

// Launch ecs instance for PD and join k8s cluster
module "pd" {
  source  = "alibaba/ecs-instance/alicloud"
  version = "1.2.2"
  vswitch_id = "${var.vswich_id != "" ? var.vswitch_id : alicloud_vswitch.default.id}"
  group_ids = ["${if var.security_group_id != "" ? var.security_group_id : module.security_group.security_group_id}"]
  disk_category = "${var.worker_disk_category}"
  disk_size = "${var.worker_disk_size}"
  instance_tags = {
    app = "tidb"
  }
  host_name = "${var.cluster_name}-pd"
  instance_type = "${data.alicloud_instance_types.pd.instance_types.0.id}"
  number_of_instances = "${var.pd_count}"
  key_name = "${alicloud_key_pair.default.key_name}"
}

// Launch ecs instance for TiKV and join k8s cluster
module "tikv" {
  source  = "alibaba/ecs-instance/alicloud"
  version = "1.2.2"
  vswitch_id = "${var.vswich_id != "" ? var.vswitch_id : alicloud_vswitch.default.id}"
  group_ids = ["${if var.security_group_id != "" ? var.security_group_id : module.security_group.security_group_id}"]
  disk_category = "${var.worker_disk_category}"
  disk_size = "${var.worker_disk_size}"
  instance_tags = {
    app = "tidb"
  }
  host_name = "${var.cluster_name}-tikv"
  instance_type = "${data.alicloud_instance_types.tikv.instance_types.0.id}"
  number_of_instances = "${var.tikv_count}"
  key_name = "${alicloud_key_pair.default.key_name}"
}



