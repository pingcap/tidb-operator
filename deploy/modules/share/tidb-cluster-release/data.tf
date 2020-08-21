data "external" "tidb_hostname" {
  count       = var.create ? 1 : 0
  depends_on  = [helm_release.tidb-cluster, null_resource.wait-lb-ip]
  working_dir = path.cwd
  program     = ["bash", "-c", "kubectl --kubeconfig ${var.kubeconfig_filename} get svc -n ${var.cluster_name} ${var.cluster_name}-tidb -o json | jq '.status.loadBalancer.ingress[0]'"]
}

data "external" "monitor_hostname" {
  count       = var.create ? 1 : 0
  depends_on  = [helm_release.tidb-cluster, null_resource.wait-mlb-ip]
  working_dir = path.cwd
  program     = ["bash", "-c", "kubectl --kubeconfig ${var.kubeconfig_filename} get svc -n ${var.cluster_name} ${var.cluster_name}-grafana -o json | jq '.status.loadBalancer.ingress[0]'"]
}

data "external" "tidb_port" {
  count       = var.create ? 1 : 0
  depends_on  = [helm_release.tidb-cluster]
  working_dir = path.cwd
  program     = ["bash", "-c", "kubectl --kubeconfig ${var.kubeconfig_filename} get svc -n ${var.cluster_name} ${var.cluster_name}-tidb -o json | jq '.spec.ports | .[] | select( .name == \"mysql-client\") | {port: .port|tostring}'"]
}

data "external" "monitor_port" {
  count       = var.create ? 1 : 0
  depends_on  = [helm_release.tidb-cluster]
  working_dir = path.cwd
  program     = ["bash", "-c", "kubectl --kubeconfig ${var.kubeconfig_filename} get svc -n ${var.cluster_name} ${var.cluster_name}-grafana -o json | jq '.spec.ports | .[] | select( .name == \"grafana\") | {port: .port|tostring}'"]
}
