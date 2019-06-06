data "aws_availability_zones" "available" {}

data "aws_ami" "amazon-linux-2" {
 most_recent = true

 owners = ["amazon"]

 filter {
   name   = "name"
   values = ["amzn2-ami-hvm-*-x86_64-gp2"]
 }
}

data "template_file" "tidb_cluster_values" {
  template = "${file("${path.module}/templates/tidb-cluster-values.yaml.tpl")}"
  vars  {
    cluster_version = "${var.tidb_version}"
    pd_replicas = "${var.pd_count}"
    tikv_replicas = "${var.tikv_count}"
    tidb_replicas = "${var.tidb_count}"
    monitor_enable_anonymous_user = "${var.monitor_enable_anonymous_user}"
  }
}

# kubernetes provider can't use computed config_path right now, see issue:
# https://github.com/terraform-providers/terraform-provider-kubernetes/issues/142
# so we don't use kubernetes provider to retrieve tidb and monitor connection info,
# instead we use external data source.
# data "kubernetes_service" "tidb" {
#   depends_on = ["helm_release.tidb-cluster"]
#   metadata {
#     name = "tidb-cluster-${var.cluster_name}-tidb"
#     namespace = "tidb"
#   }
# }

# data "kubernetes_service" "monitor" {
#   depends_on = ["helm_release.tidb-cluster"]
#   metadata {
#     name = "tidb-cluster-${var.cluster_name}-grafana"
#     namespace = "tidb"
#   }
# }

data "external" "tidb_service" {
  depends_on = ["null_resource.wait-tidb-ready"]
  program = ["bash", "-c", "kubectl --kubeconfig credentials/kubeconfig_${var.cluster_name} get svc -n tidb tidb-cluster-${var.cluster_name}-tidb -ojson | jq '.status.loadBalancer.ingress[0]'"]
}

data "external" "monitor_service" {
  depends_on = ["null_resource.wait-tidb-ready"]
  program = ["bash", "-c", "kubectl --kubeconfig credentials/kubeconfig_${var.cluster_name} get svc -n tidb tidb-cluster-${var.cluster_name}-grafana -ojson | jq '.status.loadBalancer.ingress[0]'"]
}
