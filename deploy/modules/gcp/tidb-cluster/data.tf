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

locals {
  # examples of location: us-central1 (region), us-central1-b (zone), us-central1-c (zone)
  cluster_location_args = "%{if length(split("-", var.gke_cluster_location)) == 3}--zone ${var.gke_cluster_location} %{else}--region ${var.gke_cluster_location} %{endif}"
  # TODO Update related code when node locations is avaiable in attributes of cluster resource.
  cmd_get_cluster_locations = <<EOT
gcloud --project ${var.gcp_project} container clusters list --filter='name=${var.gke_cluster_name}' --format='json[no-heading](locations)' ${local.cluster_location_args} | jq '{"locations": (if (. | length) > 0 then .[0].locations | join(",") else "" end) }'
EOT
}

data "external" "cluster_locations" {
  depends_on = [var.cluster_id]
  program    = ["bash", "-c", local.cmd_get_cluster_locations]
}
