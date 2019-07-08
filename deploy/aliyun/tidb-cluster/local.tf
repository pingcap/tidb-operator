locals {

  group_default = {
    min_size = 2
    max_size = 2
  }

  tidb_cluster_worker_groups = [
    {
      name = "${var.cluster_name}-pd"
      key_name = var.key_name
      instance_type = var.pd_instance_type
      min_size = var.pd_count
      max_size = var.pd_count
      node_taints = "dedicated=pd:NoSchedule"
      node_labels = "dedicated=pd"
      post_userdata = file("userdata.sh")
    },
    {
      name = "${var.cluster_name}-tikv"
      key_name = var.key_name
      instance_type = var.tikv_instance_type
      min_size = var.tikv_count
      max_size = var.tikv_count
      node_taints = "dedicated=tikv:NoSchedule"
      node_labels = "dedicated=tikv"
      post_userdata = file("userdata.sh")
    },
    {
      name = "${var.cluster_name}-tidb"
      key_name = var.key_name
      instance_type = var.tidb_instance_type
      min_size = var.tidb_count
      max_size = var.tidb_count
      node_taints = "dedicated=tidb:NoSchedule"
      node_labels = "dedicated=tidb"
    },
    {
      name = "${var.cluster_name}-monitor"
      key_name = var.key_name
      instance_type = var.monitor_instance_type
      min_size = 1
      max_size = 1
    }
  ]
}