data "template_file" "tidb_cluster_values" {
  template = "${file("${path.module}/templates/tidb-cluster-values.yaml.tpl")}"

  vars {
    cluster_version = "${var.tidb_version}"
    pd_replicas     = "${var.pd_replica_count}"
    tikv_replicas   = "${var.tikv_replica_count}"
    tidb_replicas   = "${var.tidb_replica_count}"
  }
}

data "external" "tidb_ilb_ip" {
  depends_on = ["null_resource.deploy-tidb-cluster"]
  program    = ["bash", "-c", "kubectl --kubeconfig ${local.kubeconfig} get svc -n tidb tidb-cluster-tidb -o json | jq '.status.loadBalancer.ingress[0]'"]
}
