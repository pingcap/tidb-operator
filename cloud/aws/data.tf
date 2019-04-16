data "aws_availability_zones" "available" {}

data "template_file" "tidb_cluster_values" {
  template = "${file("${path.module}/templates/tidb-cluster-values.yaml.tpl")}"
  vars  {
    cluster_version = "${var.tidb_version}"
    pd_replicas = "${var.pd_count}"
    tikv_replicas = "${var.tikv_count}"
    tidb_replicas = "${var.tidb_count}"
  }
}

data "kubernetes_service" "tidb" {
  depends_on = ["helm_release.tidb-cluster"]
  metadata {
    name = "tidb-cluster-tidb"
    namespace = "tidb"
  }
}

data "kubernetes_service" "monitor" {
  depends_on = ["helm_release.tidb-cluster"]
  metadata {
    name = "tidb-cluster-grafana"
    namespace = "tidb"
  }
}
