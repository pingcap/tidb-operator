data "template_file" "userdata" {
  template = file("${path.module}/templates/user_data.sh.tpl")
  count    = length(local.tidb_cluster_worker_groups)

  vars = {
    pre_userdata = lookup(
      local.tidb_cluster_worker_groups[count.index],
      "pre_userdata",
      local.group_default.pre_userdata
    )
    post_userdata = lookup(
      local.tidb_cluster_worker_groups[count.index],
      "post_userdata",
      local.group_default.post_userdata
    )
    open_api_token = var.ack.bootstrap_token
    node_taints = lookup(
      local.tidb_cluster_worker_groups[count.index],
      "node_taints",
      local.group_default.node_taints
    )
    node_labels = lookup(
      local.tidb_cluster_worker_groups[count.index],
      "node_labels",
      local.group_default.node_labels
    )
    region = var.ack.region
  }
}

resource "alicloud_ess_scaling_group" "workers" {
  count              = length(local.tidb_cluster_worker_groups)
  scaling_group_name = "${var.ack.cluster_name}-${lookup(local.tidb_cluster_worker_groups[count.index], "name", count.index)}"
  vswitch_ids        = var.ack.vswitch_ids
  min_size = lookup(
    local.tidb_cluster_worker_groups[count.index],
    "min_size",
    local.group_default.min_size,
  )
  max_size = lookup(
    local.tidb_cluster_worker_groups[count.index],
    "max_size",
    local.group_default.max_size,
  )
  default_cooldown = lookup(
    local.tidb_cluster_worker_groups[count.index],
    "default_cooldown",
    local.group_default["default_cooldown"]
  )
  multi_az_policy = "BALANCE"

  removal_policies = [
    "OldestScalingConfiguration",
    "NewestInstance",
  ]

  lifecycle {
    ignore_changes = [vswitch_ids]
  }
}

# Create the corresponding auto-scaling configurations
resource "alicloud_ess_scaling_configuration" "workers" {
  count                      = length(local.tidb_cluster_worker_groups)
  scaling_group_id           = element(alicloud_ess_scaling_group.workers.*.id, count.index)
  instance_type              = local.tidb_cluster_worker_groups[count.index].instance_type
  image_id                   = var.image_id
  security_group_id          = var.ack.security_group_id
  key_name                   = var.ack.key_name
  instance_name              = local.tidb_cluster_worker_groups[count.index].name
  user_data                  = element(data.template_file.userdata.*.rendered, count.index)
  system_disk_category       = lookup(local.tidb_cluster_worker_groups[count.index], "system_disk_category", local.group_default["system_disk_category"])
  system_disk_size           = lookup(local.tidb_cluster_worker_groups[count.index], "system_disk_size", local.group_default["system_disk_size"])
  internet_charge_type       = lookup(local.tidb_cluster_worker_groups[count.index], "internet_charge_type", local.group_default["internet_charge_type"])
  internet_max_bandwidth_in  = lookup(local.tidb_cluster_worker_groups[count.index], "internet_max_bandwidth_in", local.group_default["internet_max_bandwidth_in"])
  internet_max_bandwidth_out = lookup(local.tidb_cluster_worker_groups[count.index], "internet_max_bandwidth_out", local.group_default["internet_max_bandwidth_out"])

  enable       = true
  active       = true
  force_delete = true

  tags = {
    name                                                = "${var.ack.cluster_name}-${lookup(local.tidb_cluster_worker_groups[count.index], "name", count.index)}-ack_asg"
    "kubernetes.io/cluster/${var.ack.cluster_name}"     = "owned"
    "k8s.io/cluster-autoscaler/${var.ack.cluster_name}" = "default"
  }

  lifecycle {
    ignore_changes = [instance_type]
  }
}
