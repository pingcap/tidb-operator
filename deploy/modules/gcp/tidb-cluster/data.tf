data "external" "tidb_ilb_ip" {
  depends_on = [null_resource.wait-lb-ip]
  program    = ["bash", "-c", local.cmd_get_tidb_ilb_ip]
}

data "external" "monitor_lb_ip" {
  depends_on = [null_resource.wait-lb-ip]
  program    = ["bash", "-c", local.cmd_get_monitor_lb_ip]
}

data "external" "tidb_port" {
  depends_on = [null_resource.wait-lb-ip]
  program    = ["bash", "-c", local.cmd_get_tidb_port]
}

data "external" "monitor_port" {
  depends_on = [null_resource.wait-lb-ip]
  program    = ["bash", "-c", local.cmd_get_monitor_port]
}

locals {
  # examples of location: us-central1 (region), us-central1-b (zone), us-central1-c (zone)
  cluster_location_args = "%{if length(split("-", var.gke_cluster_location)) == 3}--zone ${var.gke_cluster_location} %{else}--region ${var.gke_cluster_location} %{endif}"
  # TODO Update related code when node locations is avaiable in attributes of cluster resource.
  cmd_get_cluster_locations = <<EOT
gcloud --project ${var.gcp_project} container clusters list --filter='name=${var.gke_cluster_name}' --format='json[no-heading](locations)' ${local.cluster_location_args} | jq '{"locations": (if (. | length) > 0 then .[0].locations | join(",") else "" end) }'
EOT
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

data "external" "cluster_locations" {
  depends_on = [var.cluster_id]
  program    = ["bash", "-c", local.cmd_get_cluster_locations]
}
