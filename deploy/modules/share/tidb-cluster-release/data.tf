data "external" "tidb_hostname" {
  depends_on  = [helm_release.tidb-cluster]
  working_dir = path.cwd
  program     = ["bash", "-c", "kubectl --kubeconfig ${var.kubeconfig_filename} get svc -n ${var.cluster_name} ${var.cluster_name}-tidb -o json | jq '.status.loadBalancer.ingress[0]'"]
}

data "external" "monitor_hostname" {
  depends_on  = [helm_release.tidb-cluster]
  working_dir = path.cwd
  program     = ["bash", "-c", "kubectl --kubeconfig ${var.kubeconfig_filename} get svc -n ${var.cluster_name} ${var.cluster_name}-grafana -o json | jq '.status.loadBalancer.ingress[0]'"]
}
