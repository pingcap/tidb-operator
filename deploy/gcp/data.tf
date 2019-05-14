data "template_file" "tidb_cluster_values" {
  template = "${file("${path.module}/templates/tidb-cluster-values.yaml.tpl")}"
  vars  {
    cluster_version = "${var.tidb_version}"
    pd_replicas = "${var.pd_count}"
    tikv_replicas = "${var.tikv_count}"
    tidb_replicas = "${var.tidb_count}"
  }
}
