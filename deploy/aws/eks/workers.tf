# Worker Groups using Launch Configurations

resource "aws_autoscaling_group" "workers" {
  name_prefix = "${aws_eks_cluster.this.name}-${lookup(local.control_worker_groups[count.index], "name", count.index)}"
  desired_capacity = lookup(
    local.control_worker_groups[count.index],
    "asg_desired_capacity",
    local.workers_group_defaults["asg_desired_capacity"],
  )
  max_size = lookup(
    local.control_worker_groups[count.index],
    "asg_max_size",
    local.workers_group_defaults["asg_max_size"],
  )
  min_size = lookup(
    local.control_worker_groups[count.index],
    "asg_min_size",
    local.workers_group_defaults["asg_min_size"],
  )
  force_delete = lookup(
    local.control_worker_groups[count.index],
    "asg_force_delete",
    local.workers_group_defaults["asg_force_delete"],
  )
  # target_group_arns = compact(
  #   split(
  #     ",",
  #     coalesce(
  #       lookup(local.control_worker_groups[count.index], "target_group_arns", ""),
  #       local.workers_group_defaults["target_group_arns"],
  #     ),
  #   ),
  # )
  launch_configuration = element(aws_launch_configuration.workers.*.id, count.index)
  vpc_zone_identifier = split(
    ",",
    coalesce(
      lookup(local.control_worker_groups[count.index], "subnets", ""),
      local.workers_group_defaults["subnets"],
    ),
  )
  protect_from_scale_in = lookup(
    local.control_worker_groups[count.index],
    "protect_from_scale_in",
    local.workers_group_defaults["protect_from_scale_in"],
  )
  # suspended_processes = compact(
  #   split(
  #     ",",
  #     coalesce(
  #       lookup(local.control_worker_groups[count.index], "suspended_processes", ""),
  #       local.workers_group_defaults["suspended_processes"],
  #     ),
  #   ),
  # )
  # enabled_metrics = compact(
  #   split(
  #     ",",
  #     coalesce(
  #       lookup(local.control_worker_groups[count.index], "enabled_metrics", ""),
  #       local.workers_group_defaults["enabled_metrics"],
  #     ),
  #   ),
  # )
  count = var.worker_group_count
  placement_group = lookup(
    local.control_worker_groups[count.index],
    "placement_group",
    local.workers_group_defaults["placement_group"],
  )

  tags = concat(
    [
      {
        key = "Name"
        value = "${aws_eks_cluster.this.name}-${lookup(local.control_worker_groups[count.index], "name", count.index)}-eks_asg"
        propagate_at_launch = true
      },
      {
        key = "kubernetes.io/cluster/${aws_eks_cluster.this.name}"
        value = "owned"
        propagate_at_launch = true
      },
      {
        key = "k8s.io/cluster-autoscaler/${lookup(
          local.control_worker_groups[count.index],
          "autoscaling_enabled",
          local.workers_group_defaults["autoscaling_enabled"],
        ) == 1 ? "enabled" : "disabled"}"
        value = "true"
        propagate_at_launch = false
      },
      # {
      #   "key"                 = "k8s.io/cluster-autoscaler/${aws_eks_cluster.this.name}"
      #   "value"               = ""
      #   "propagate_at_launch" = false
      # },
      {
        key = "k8s.io/cluster-autoscaler/node-template/resources/ephemeral-storage"
        value = "${lookup(
          local.control_worker_groups[count.index],
          "root_volume_size",
          local.workers_group_defaults["root_volume_size"],
        )}Gi"
        propagate_at_launch = false
      },
    ],
    local.asg_tags,
    var.worker_group_tags[contains(
      keys(var.worker_group_tags),
      lookup(local.control_worker_groups[count.index], "name", count.index),
    ) ? lookup(local.control_worker_groups[count.index], "name", count.index) : "default"],
  )


  lifecycle {
    create_before_destroy = true
    # ignore_changes = ["desired_capacity"]
  }
}

resource "aws_launch_configuration" "workers" {
  name_prefix = "${aws_eks_cluster.this.name}-${lookup(local.control_worker_groups[count.index], "name", count.index)}"
  associate_public_ip_address = lookup(
    local.control_worker_groups[count.index],
    "public_ip",
    local.workers_group_defaults["public_ip"],
  )
  security_groups = concat([local.worker_security_group_id], var.worker_additional_security_group_ids, compact(
    split(
      ",",
      lookup(
        local.control_worker_groups[count.index],
        "additional_security_group_ids",
        local.workers_group_defaults["additional_security_group_ids"],
      ),
    ),
  ))
  iam_instance_profile = element(aws_iam_instance_profile.workers.*.id, count.index)
  image_id = lookup(
    local.control_worker_groups[count.index],
    "ami_id",
    local.workers_group_defaults["ami_id"],
  )
  instance_type = lookup(
    local.control_worker_groups[count.index],
    "instance_type",
    local.workers_group_defaults["instance_type"],
  )
  key_name = lookup(
    local.control_worker_groups[count.index],
    "key_name",
    local.workers_group_defaults["key_name"],
  )
  user_data_base64 = base64encode(element(data.template_file.userdata.*.rendered, count.index))
  ebs_optimized = lookup(
    local.control_worker_groups[count.index],
    "ebs_optimized",
    lookup(
      local.ebs_optimized,
      lookup(
        local.control_worker_groups[count.index],
        "instance_type",
        local.workers_group_defaults["instance_type"],
      ),
      false,
    ),
  )
  enable_monitoring = lookup(
    local.control_worker_groups[count.index],
    "enable_monitoring",
    local.workers_group_defaults["enable_monitoring"],
  )
  spot_price = lookup(
    local.control_worker_groups[count.index],
    "spot_price",
    local.workers_group_defaults["spot_price"],
  )
  placement_tenancy = lookup(
    local.control_worker_groups[count.index],
    "placement_tenancy",
    local.workers_group_defaults["placement_tenancy"],
  )
  count = var.worker_group_count

  lifecycle {
    create_before_destroy = true
  }

  root_block_device {
    volume_size = lookup(
      local.control_worker_groups[count.index],
      "root_volume_size",
      local.workers_group_defaults["root_volume_size"],
    )
    volume_type = lookup(
      local.control_worker_groups[count.index],
      "root_volume_type",
      local.workers_group_defaults["root_volume_type"],
    )
    iops = lookup(
      local.control_worker_groups[count.index],
      "root_iops",
      local.workers_group_defaults["root_iops"],
    )
    delete_on_termination = true
  }
}

resource "aws_security_group" "workers" {
  name_prefix = aws_eks_cluster.this.name
  description = "Security group for all nodes in the cluster."
  vpc_id      = var.vpc_id
  count       = var.worker_create_security_group ? 1 : 0
  tags = merge(
    var.tags,
    {
      Name = "${aws_eks_cluster.this.name}-eks_worker_sg"
      "kubernetes.io/cluster/${aws_eks_cluster.this.name}" = "owned"
    },
  )
}

resource "aws_security_group_rule" "workers_egress_internet" {
  description       = "Allow nodes all egress to the Internet."
  protocol          = "-1"
  security_group_id = aws_security_group.workers[0].id
  cidr_blocks       = ["0.0.0.0/0"]
  from_port         = 0
  to_port           = 0
  type              = "egress"
  count             = var.worker_create_security_group ? 1 : 0
}

resource "aws_security_group_rule" "workers_ingress_self" {
  description              = "Allow node to communicate with each other."
  protocol                 = "-1"
  security_group_id        = aws_security_group.workers[0].id
  source_security_group_id = aws_security_group.workers[0].id
  from_port                = 0
  to_port                  = 65535
  type                     = "ingress"
  count                    = var.worker_create_security_group ? 1 : 0
}

resource "aws_security_group_rule" "workers_ingress_cluster" {
  description              = "Allow workers pods to receive communication from the cluster control plane."
  protocol                 = "tcp"
  security_group_id        = aws_security_group.workers[0].id
  source_security_group_id = local.cluster_security_group_id
  from_port                = var.worker_sg_ingress_from_port
  to_port                  = 65535
  type                     = "ingress"
  count                    = var.worker_create_security_group ? 1 : 0
}

resource "aws_security_group_rule" "workers_ingress_cluster_kubelet" {
  description              = "Allow workers Kubelets to receive communication from the cluster control plane."
  protocol                 = "tcp"
  security_group_id        = aws_security_group.workers[0].id
  source_security_group_id = local.cluster_security_group_id
  from_port                = 10250
  to_port                  = 10250
  type                     = "ingress"
  count                    = var.worker_create_security_group ? var.worker_sg_ingress_from_port > 10250 ? 1 : 0 : 0
}

resource "aws_security_group_rule" "workers_ingress_cluster_https" {
  description              = "Allow pods running extension API servers on port 443 to receive communication from cluster control plane."
  protocol                 = "tcp"
  security_group_id        = aws_security_group.workers[0].id
  source_security_group_id = local.cluster_security_group_id
  from_port                = 443
  to_port                  = 443
  type                     = "ingress"
  count                    = var.worker_create_security_group ? 1 : 0
}

resource "aws_iam_role" "workers" {
  name_prefix           = aws_eks_cluster.this.name
  assume_role_policy    = data.aws_iam_policy_document.workers_assume_role_policy.json
  permissions_boundary  = var.permissions_boundary
  path                  = var.iam_path
  force_detach_policies = true
}

resource "aws_iam_instance_profile" "workers" {
  name_prefix = aws_eks_cluster.this.name
  role = lookup(
    local.control_worker_groups[count.index],
    "iam_role_id",
    local.workers_group_defaults["iam_role_id"],
  )
  count = var.worker_group_count
  path  = var.iam_path
}

resource "aws_iam_role_policy_attachment" "workers_AmazonEKSWorkerNodePolicy" {
  policy_arn = "arn:aws:iam::aws:policy/AmazonEKSWorkerNodePolicy"
  role       = aws_iam_role.workers.name
}

resource "aws_iam_role_policy_attachment" "workers_AmazonEKS_CNI_Policy" {
  policy_arn = "arn:aws:iam::aws:policy/AmazonEKS_CNI_Policy"
  role       = aws_iam_role.workers.name
}

resource "aws_iam_role_policy_attachment" "workers_AmazonEC2ContainerRegistryReadOnly" {
  policy_arn = "arn:aws:iam::aws:policy/AmazonEC2ContainerRegistryReadOnly"
  role       = aws_iam_role.workers.name
}

resource "aws_iam_role_policy_attachment" "workers_additional_policies" {
  count      = var.workers_additional_policies_count
  role       = aws_iam_role.workers.name
  policy_arn = var.workers_additional_policies[count.index]
}

resource "null_resource" "tags_as_list_of_maps" {
  count = length(keys(var.tags))

  triggers = {
    key                 = element(keys(var.tags), count.index)
    value               = element(values(var.tags), count.index)
    propagate_at_launch = "true"
  }
}

resource "aws_iam_role_policy_attachment" "workers_autoscaling" {
  policy_arn = aws_iam_policy.worker_autoscaling.arn
  role       = aws_iam_role.workers.name
}

resource "aws_iam_policy" "worker_autoscaling" {
  name_prefix = "eks-worker-autoscaling-${aws_eks_cluster.this.name}"
  description = "EKS worker node autoscaling policy for cluster ${aws_eks_cluster.this.name}"
  policy      = data.aws_iam_policy_document.worker_autoscaling.json
  path        = var.iam_path
}

data "aws_iam_policy_document" "worker_autoscaling" {
  statement {
    sid    = "eksWorkerAutoscalingAll"
    effect = "Allow"

    actions = [
      "autoscaling:DescribeAutoScalingGroups",
      "autoscaling:DescribeAutoScalingInstances",
      "autoscaling:DescribeLaunchConfigurations",
      "autoscaling:DescribeTags",
      "ec2:DescribeLaunchTemplateVersions",
    ]

    resources = ["*"]
  }

  statement {
    sid    = "eksWorkerAutoscalingOwn"
    effect = "Allow"

    actions = [
      "autoscaling:SetDesiredCapacity",
      "autoscaling:TerminateInstanceInAutoScalingGroup",
      "autoscaling:UpdateAutoScalingGroup",
    ]

    resources = ["*"]

    condition {
      test     = "StringEquals"
      variable = "autoscaling:ResourceTag/kubernetes.io/cluster/${aws_eks_cluster.this.name}"
      values   = ["owned"]
    }

    condition {
      test     = "StringEquals"
      variable = "autoscaling:ResourceTag/k8s.io/cluster-autoscaler/enabled"
      values   = ["true"]
    }
  }
}
