data "template_file" "tidb-cluster-values" {
  template = file("${path.module}/templates/tidb-cluster-values.yaml.tpl")

  vars = {
    cluster_version     = var.tidb_version
    pd_replicas         = var.pd_count
    tikv_replicas       = var.tikv_count
    tidb_replicas       = var.tidb_count
    local_storage_class = "local-volume"
    pd_storage_size = "${floor(
      data.alicloud_instance_types.pd.instance_types[0].local_storage.capacity * 0.98,
    )}Gi"
    tikv_storage_size = "${floor(
      data.alicloud_instance_types.tikv.instance_types[0].local_storage.capacity * 0.98,
    )}Gi"
    monitor_storage_class           = var.monitor_storage_class
    monitor_storage_size            = "${var.monitor_storage_size}Gi"
    tikv_defaultcf_block_cache_size = "${var.tikv_memory_size * 0.4}GB"
    tikv_writecf_block_cache_size   = "${var.tikv_memory_size * 0.2}GB"
    monitor_reserve_days            = var.monitor_reserve_days
    monitor_slb_network_type        = var.monitor_slb_network_type
    monitor_enable_anonymous_user   = var.monitor_enable_anonymous_user
  }
}

data "template_file" "local-volume-provisioner" {
  template = file("${path.module}/templates/local-volume-provisioner.yaml.tpl")

  vars = {
    access_key_id     = var.ALICLOUD_ACCESS_KEY
    access_key_secret = var.ALICLOUD_SECRET_KEY
  }
}

data "alicloud_instance_types" "pd" {
  provider             = alicloud.this
  instance_type_family = var.pd_instance_type_family
  memory_size          = var.pd_instance_memory_size
  network_type         = "Vpc"
}

data "alicloud_instance_types" "tikv" {
  provider             = alicloud.this
  instance_type_family = var.tikv_instance_type_family
  memory_size          = var.tikv_memory_size
  network_type         = "Vpc"
}

data "alicloud_instance_types" "tidb" {
  provider       = alicloud.this
  cpu_core_count = var.tidb_instance_core_count
  memory_size    = var.tidb_instance_memory_size
  network_type   = "Vpc"
}

data "alicloud_instance_types" "monitor" {
  provider       = alicloud.this
  cpu_core_count = var.monitor_instance_core_count
  memory_size    = var.monitor_instance_memory_size
  network_type   = "Vpc"
}

data "external" "tidb_slb_ip" {
  depends_on = [null_resource.wait-tidb-ready]
  program    = ["bash", "-c", "kubectl --kubeconfig ${local.kubeconfig} get svc -n tidb tidb-cluster-tidb -o json | jq '.status.loadBalancer.ingress[0]'"]
}

data "external" "monitor_slb_ip" {
  depends_on = [null_resource.wait-tidb-ready]
  program    = ["bash", "-c", "kubectl --kubeconfig ${local.kubeconfig} get svc -n tidb tidb-cluster-grafana -o json | jq '.status.loadBalancer.ingress[0]'"]
}

data "external" "tidb_port" {
  depends_on = [null_resource.wait-tidb-ready]
  program    = ["bash", "-c", "kubectl --kubeconfig ${local.kubeconfig} get svc -n tidb tidb-cluster-tidb -o json | jq '.spec.ports | .[] | select( .name == \"mysql-client\") | {port: .port|tostring}'"]
}

data "external" "monitor_port" {
  depends_on = [null_resource.wait-tidb-ready]
  program    = ["bash", "-c", "kubectl --kubeconfig ${local.kubeconfig} get svc -n tidb tidb-cluster-grafana -o json | jq '.spec.ports | .[] | select( .name == \"grafana\") | {port: .port|tostring}'"]
}

