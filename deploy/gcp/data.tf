data "template_file" "tidb_cluster_values" {
  template = file("${path.module}/templates/tidb-cluster-values.yaml.tpl")

  vars = {
    cluster_version  = var.tidb_version
    pd_replicas      = var.pd_replica_count
    tikv_replicas    = var.tikv_replica_count
    tidb_replicas    = var.tidb_replica_count
    operator_version = var.tidb_operator_version
    tidb_operator_registry = var.tidb_operator_registry
  }
}

data "google_compute_zones" "available" { }

data "external" "tidb_ilb_ip" {
  depends_on = [null_resource.deploy-tidb-cluster]
  program    = ["bash", "-c", "kubectl --kubeconfig ${local.kubeconfig} get svc -n tidb tidb-cluster-tidb -o json | jq '.status.loadBalancer.ingress[0]'"]
}

data "external" "monitor_ilb_ip" {
  depends_on = [null_resource.deploy-tidb-cluster]
  program    = ["bash", "-c", "kubectl --kubeconfig ${local.kubeconfig} get svc -n tidb tidb-cluster-grafana -o json | jq '.status.loadBalancer.ingress[0]'"]
}

data "external" "tidb_port" {
  depends_on = [null_resource.deploy-tidb-cluster]
  program    = ["bash", "-c", "kubectl --kubeconfig ${local.kubeconfig} get svc -n tidb tidb-cluster-tidb -o json | jq '.spec.ports | .[] | select( .name == \"mysql-client\") | {port: .port|tostring}'"]
}

data "external" "monitor_port" {
  depends_on = [null_resource.deploy-tidb-cluster]
  program    = ["bash", "-c", "kubectl --kubeconfig ${local.kubeconfig} get svc -n tidb tidb-cluster-grafana -o json | jq '.spec.ports | .[] | select( .name == \"grafana\") | {port: .port|tostring}'"]
}

