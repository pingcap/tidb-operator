data "external" "tidb_ilb_ip" {
  depends_on = [module.default-tidb-cluster]
  program    = ["bash", "-c", "kubectl --kubeconfig ${local.kubeconfig} get svc -n ${var.default_tidb_cluster_name} ${var.default_tidb_cluster_name}-tidb -o json | jq '.status.loadBalancer.ingress[0]'"]
}

data "external" "monitor_ilb_ip" {
  depends_on = [module.default-tidb-cluster]
  program    = ["bash", "-c", "kubectl --kubeconfig ${local.kubeconfig} get svc -n ${var.default_tidb_cluster_name} ${var.default_tidb_cluster_name}-grafana -o json | jq '.status.loadBalancer.ingress[0]'"]
}

data "external" "tidb_port" {
  depends_on = [module.default-tidb-cluster]
  program    = ["bash", "-c", "kubectl --kubeconfig ${local.kubeconfig} get svc -n ${var.default_tidb_cluster_name} ${var.default_tidb_cluster_name}-tidb -o json | jq '.spec.ports | .[] | select( .name == \"mysql-client\") | {port: .port|tostring}'"]
}

data "external" "monitor_port" {
  depends_on = [module.default-tidb-cluster]
  program    = ["bash", "-c", "kubectl --kubeconfig ${local.kubeconfig} get svc -n ${var.default_tidb_cluster_name} ${var.default_tidb_cluster_name}-grafana -o json | jq '.spec.ports | .[] | select( .name == \"grafana\") | {port: .port|tostring}'"]
}

