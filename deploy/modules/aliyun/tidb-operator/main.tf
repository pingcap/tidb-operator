# Alicloud ACK module launches a managed kubernetes cluster
resource "alicloud_key_pair" "default" {
  count           = var.key_pair_name == "" ? 1 : 0
  key_name_prefix = "${var.cluster_name}-key"
  key_file        = var.key_file != "" ? var.key_file : format("%s/%s-key", path.module, var.cluster_name)
}

# If there is not specifying vpc_id, create a new one
resource "alicloud_vpc" "vpc" {
  count      = var.vpc_id == "" ? 1 : 0
  cidr_block = var.vpc_cidr
  name       = "${var.cluster_name}-vpc"

  lifecycle {
    ignore_changes = [cidr_block]
  }
}

# For new vpc or existing vpc with no vswitches, create vswitch for each zone
resource "alicloud_vswitch" "all" {
  count  = var.vpc_id != "" && length(data.alicloud_vswitches.default.vswitches) != 0 ? 0 : length(data.alicloud_zones.all.zones)
  vpc_id = alicloud_vpc.vpc[0].id
  cidr_block = cidrsubnet(
    alicloud_vpc.vpc[0].cidr_block,
    var.vpc_cidr_newbits,
    count.index,
  )
  availability_zone = data.alicloud_zones.all.zones[count.index % length(data.alicloud_zones.all.zones)]["id"]
  name              = format("vsw-%s-%d", var.cluster_name, count.index + 1)
}

resource "alicloud_security_group" "group" {
  count       = var.group_id == "" ? 1 : 0
  name        = "${var.cluster_name}-sg"
  vpc_id      = var.vpc_id != "" ? var.vpc_id : alicloud_vpc.vpc[0].id
  description = "Security group for ACK worker nodes"
}

# Allow traffic inside VPC
resource "alicloud_security_group_rule" "cluster_worker_ingress" {
  count             = var.group_id == "" ? 1 : 0
  security_group_id = alicloud_security_group.group[0].id
  type              = "ingress"
  ip_protocol       = "all"
  nic_type          = "intranet"
  port_range        = "-1/-1"
  cidr_ip           = var.vpc_id != "" ? var.vpc_cidr : alicloud_vpc.vpc[0].cidr_block
}

# Create a managed Kubernetes cluster
resource "alicloud_cs_managed_kubernetes" "k8s" {
  name = var.cluster_name
  // 'version' is a reserved parameter and it just is used to test. No Recommendation to expose it.
  // https://github.com/terraform-providers/terraform-provider-alicloud/blob/master/alicloud/resource_alicloud_cs_kubernetes.go#L396-L401
  version = var.k8s_version

  // split and join: workaround for terraform's limitation of conditional list choice, similarly hereinafter
  vswitch_ids = [
    element(
      split(
        ",",
        var.vpc_id != "" && length(data.alicloud_vswitches.default.vswitches) != 0 ? join(",", data.template_file.vswitch_id.*.rendered) : join(",", alicloud_vswitch.all.*.id),
      ),
      0,
  )]
  key_name              = alicloud_key_pair.default[0].key_name
  pod_cidr              = var.k8s_pod_cidr
  service_cidr          = var.k8s_service_cidr
  new_nat_gateway       = var.create_nat_gateway
  cluster_network_type  = var.cluster_network_type
  slb_internet_enabled  = var.public_apiserver
  kube_config           = var.kubeconfig_file != "" ? var.kubeconfig_file : format("%s/kubeconfig", path.cwd)
  worker_number         = var.default_worker_count
  worker_instance_types = [var.default_worker_type != "" ? var.default_worker_type : data.alicloud_instance_types.default.instance_types[0].id]

  # These varialbes are 'ForceNew' that will cause kubernetes cluster re-creation
  # on variable change, so we make all these variables immutable in favor of safety.
  lifecycle {
    ignore_changes = [
      vswitch_ids,
      worker_instance_types,
      key_name,
      pod_cidr,
      service_cidr,
      cluster_network_type,
    ]
  }
}
