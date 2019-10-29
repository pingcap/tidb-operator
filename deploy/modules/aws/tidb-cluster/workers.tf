# Worker Groups using Launch Configurations

resource "aws_autoscaling_group" "workers" {
  name_prefix = "${var.eks.cluster_id}-${lookup(local.tidb_cluster_worker_groups[count.index], "name", count.index)}"
  desired_capacity = lookup(
    local.tidb_cluster_worker_groups[count.index],
    "asg_desired_capacity",
    local.workers_group_defaults["asg_desired_capacity"],
  )
  max_size = lookup(
    local.tidb_cluster_worker_groups[count.index],
    "asg_max_size",
    local.workers_group_defaults["asg_max_size"],
  )
  min_size = lookup(
    local.tidb_cluster_worker_groups[count.index],
    "asg_min_size",
    local.workers_group_defaults["asg_min_size"],
  )
  force_delete         = false
  launch_configuration = element(aws_launch_configuration.workers.*.id, count.index)
  vpc_zone_identifier = split(
    ",",
    coalesce(
      lookup(local.tidb_cluster_worker_groups[count.index], "subnets", ""),
      local.workers_group_defaults["subnets"],
    ),
  )
  protect_from_scale_in = false
  count                 = local.worker_group_count
  placement_group       = "" # The name of the placement group into which to launch the instances, if any.
  suspended_processes   = lookup(local.tidb_cluster_worker_groups[count.index], "suspended_processes", [])

  tags = concat(
    [
      {
        key                 = "Name"
        value               = "${var.eks.cluster_id}-${lookup(local.tidb_cluster_worker_groups[count.index], "name", count.index)}-eks_asg"
        propagate_at_launch = true
      },
      {
        key                 = "kubernetes.io/cluster/${var.eks.cluster_id}"
        value               = "owned"
        propagate_at_launch = true
      },
      {
        key = "k8s.io/cluster-autoscaler/${lookup(
          local.tidb_cluster_worker_groups[count.index],
          "autoscaling_enabled",
          local.workers_group_defaults["autoscaling_enabled"],
        ) == 1 ? "enabled" : "disabled"}"
        value               = "true"
        propagate_at_launch = false
      },
      {
        key = "k8s.io/cluster-autoscaler/node-template/resources/ephemeral-storage"
        value = "${lookup(
          local.tidb_cluster_worker_groups[count.index],
          "root_volume_size",
          local.workers_group_defaults["root_volume_size"],
        )}Gi"
        propagate_at_launch = false
      },
    ],
    local.asg_tags,
    var.worker_group_tags[contains(
      keys(var.worker_group_tags),
      lookup(local.tidb_cluster_worker_groups[count.index], "name", count.index),
    ) ? lookup(local.tidb_cluster_worker_groups[count.index], "name", count.index) : "default"],
  )


  lifecycle {
    create_before_destroy = true
    # ignore_changes = ["desired_capacity"]
  }
}

resource "aws_launch_configuration" "workers" {
  name_prefix = "${var.eks.cluster_id}-${lookup(local.tidb_cluster_worker_groups[count.index], "name", count.index)}"
  associate_public_ip_address = lookup(
    local.tidb_cluster_worker_groups[count.index],
    "public_ip",
    local.workers_group_defaults["public_ip"],
  )
  security_groups = concat([var.eks.worker_security_group_id], var.worker_additional_security_group_ids, compact(
    split(
      ",",
      lookup(
        local.tidb_cluster_worker_groups[count.index],
        "additional_security_group_ids",
        local.workers_group_defaults["additional_security_group_ids"],
      ),
    ),
  ))
  iam_instance_profile = element(var.eks.worker_iam_instance_profile_names, count.index)
  image_id = lookup(
    local.tidb_cluster_worker_groups[count.index],
    "ami_id",
    local.workers_group_defaults["ami_id"],
  )
  instance_type = lookup(
    local.tidb_cluster_worker_groups[count.index],
    "instance_type",
    local.workers_group_defaults["instance_type"],
  )
  key_name = lookup(
    local.tidb_cluster_worker_groups[count.index],
    "key_name",
    local.workers_group_defaults["key_name"],
  )
  user_data_base64 = base64encode(element(data.template_file.userdata.*.rendered, count.index))
  ebs_optimized = lookup(
    local.tidb_cluster_worker_groups[count.index],
    "ebs_optimized",
    lookup(
      local.ebs_optimized,
      lookup(
        local.tidb_cluster_worker_groups[count.index],
        "instance_type",
        local.workers_group_defaults["instance_type"],
      ),
      false,
    ),
  )
  enable_monitoring = lookup(
    local.tidb_cluster_worker_groups[count.index],
    "enable_monitoring",
    local.workers_group_defaults["enable_monitoring"],
  )
  spot_price = lookup(
    local.tidb_cluster_worker_groups[count.index],
    "spot_price",
    local.workers_group_defaults["spot_price"],
  )
  placement_tenancy = lookup(
    local.tidb_cluster_worker_groups[count.index],
    "placement_tenancy",
    local.workers_group_defaults["placement_tenancy"],
  )
  count = local.worker_group_count

  lifecycle {
    create_before_destroy = true
  }

  root_block_device {
    volume_size = lookup(
      local.tidb_cluster_worker_groups[count.index],
      "root_volume_size",
      local.workers_group_defaults["root_volume_size"],
    )
    volume_type = lookup(
      local.tidb_cluster_worker_groups[count.index],
      "root_volume_type",
      local.workers_group_defaults["root_volume_type"],
    )
    iops = lookup(
      local.tidb_cluster_worker_groups[count.index],
      "root_iops",
      local.workers_group_defaults["root_iops"],
    )
    delete_on_termination = true
  }
}

resource "null_resource" "tags_as_list_of_maps" {
  count = length(keys(var.tags))

  triggers = {
    key                 = element(keys(var.tags), count.index)
    value               = element(values(var.tags), count.index)
    propagate_at_launch = "true"
  }
}
