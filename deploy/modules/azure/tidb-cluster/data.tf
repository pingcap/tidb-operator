data "external" "tidb_ilb_ip" {
  depends_on = [null_resource.tidb-cluster]
  program    = ["bash", "-c", local.cmd_get_tidb_ilb_ip]
}

data "external" "monitor_lb_ip" {
  depends_on = [null_resource.tidb-cluster]
  program    = ["bash", "-c", local.cmd_get_monitor_lb_ip]
}

data "external" "tidb_port" {
  depends_on = [null_resource.tidb-cluster]
  program    = ["bash", "-c", local.cmd_get_tidb_port]
}

data "external" "monitor_port" {
  depends_on = [null_resource.tidb-cluster]
  program    = ["bash", "-c", local.cmd_get_monitor_port]
}

locals {
  cmd_get_tidb_ilb_ip = <<EOT
output=$(kubectl --kubeconfig ${var.kubeconfig_path} get svc -n ${var.cluster_name} ${var.cluster_name}-tidb -o json 2>/dev/null) || true
jq -s '.[0].status.loadBalancer.ingress[0] // {"ip":""}' <<<"$output"
EOT
  cmd_get_monitor_lb_ip = <<EOT
output=$(kubectl --kubeconfig ${var.kubeconfig_path} get svc -n ${var.cluster_name} ${var.cluster_name}-grafana -o json 2>/dev/null) || true
jq -s '.[0].status.loadBalancer.ingress[0] // {"ip":""}' <<<"$output"
EOT
  cmd_get_tidb_port = <<EOT
output=$(kubectl --kubeconfig ${var.kubeconfig_path} get svc -n ${var.cluster_name} ${var.cluster_name}-tidb -o json 2>/dev/null) || true
jq -s 'try (.[0].spec.ports | .[] | select( .name == "mysql-client") | {port: .port|tostring}) catch {"port":""}' <<<"$otuput"
EOT
  cmd_get_monitor_port = <<EOT
output=$(kubectl --kubeconfig ${var.kubeconfig_path} get svc -n ${var.cluster_name} ${var.cluster_name}-grafana -o json 2>/dev/null) || true
jq -s 'try (.[0].spec.ports | .[] | select( .name == "grafana") | {port: .port|tostring}) catch {"port":""}' <<<"$otuput"
EOT
}
