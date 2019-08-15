data "external" "tidb_ilb_ip" {
  depends_on = [null_resource.wait-lb-ip]
  program    = ["bash", "-c", "kubectl --kubeconfig ${var.kubeconfig_path} get svc -n ${var.cluster_name} ${var.cluster_name}-tidb -o json | jq '.status.loadBalancer.ingress[0]'"]
}

data "external" "monitor_lb_ip" {
  depends_on = [null_resource.wait-lb-ip]
  program    = ["bash", "-c", "kubectl --kubeconfig ${var.kubeconfig_path} get svc -n ${var.cluster_name} ${var.cluster_name}-grafana -o json | jq '.status.loadBalancer.ingress[0]'"]
}

data "external" "tidb_port" {
  depends_on = [null_resource.wait-lb-ip]
  program    = ["bash", "-c", "kubectl --kubeconfig ${var.kubeconfig_path} get svc -n ${var.cluster_name} ${var.cluster_name}-tidb -o json | jq '.spec.ports | .[] | select( .name == \"mysql-client\") | {port: .port|tostring}'"]
}

data "external" "monitor_port" {
  depends_on = [null_resource.wait-lb-ip]
  program    = ["bash", "-c", "kubectl --kubeconfig ${var.kubeconfig_path} get svc -n ${var.cluster_name} ${var.cluster_name}-grafana -o json | jq '.spec.ports | .[] | select( .name == \"grafana\") | {port: .port|tostring}'"]
}

data "google_compute_zones" "available" {}
