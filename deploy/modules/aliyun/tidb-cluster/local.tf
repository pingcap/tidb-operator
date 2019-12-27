locals {

  group_default = {
    min_size                   = 0
    max_size                   = 100
    default_cooldown           = 300
    image_id                   = var.image_id
    instance_type              = "ecs.g5.large"
    system_disk_category       = "cloud_efficiency"
    system_disk_size           = 50
    pre_userdata               = ""
    post_userdata              = ""
    internet_charge_type       = "PayByTraffic"
    internet_max_bandwidth_in  = 10
    internet_max_bandwidth_out = 10
    node_taints                = ""
    node_labels                = ""
  }

  tidb_cluster_worker_groups = [
    {
      name          = "${var.cluster_name}-pd"
      instance_type = var.pd_instance_type
      min_size      = var.pd_count
      max_size      = var.pd_count
      node_taints   = "dedicated=${var.cluster_name}-pd:NoSchedule"
      node_labels   = "dedicated=${var.cluster_name}-pd"
      post_userdata = file("${path.module}/userdata.sh")
    },
    {
      name          = "${var.cluster_name}-tikv"
      instance_type = var.tikv_instance_type
      min_size      = var.tikv_count
      max_size      = var.tikv_count
      node_taints   = "dedicated=${var.cluster_name}-tikv:NoSchedule"
      node_labels   = "dedicated=${var.cluster_name}-tikv,pingcap.com/aliyun-local-ssd=true"
      post_userdata = file("${path.module}/userdata.sh")
    },
    {
      name          = "${var.cluster_name}-tidb"
      instance_type = var.tidb_instance_type
      min_size      = var.tidb_count
      max_size      = var.tidb_count
      node_taints   = "dedicated=${var.cluster_name}-tidb:NoSchedule"
      node_labels   = "dedicated=${var.cluster_name}-tidb"
    },
    {
      name          = "${var.cluster_name}-monitor"
      instance_type = var.monitor_instance_type
      min_size      = 1
      max_size      = 1
    }
  ]
}
