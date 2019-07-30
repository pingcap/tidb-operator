module "tidb-cluster" {
  source = "../../share/tidb-cluster-release"
  cluster_name = var.cluster_name
  pd_count = var.pd_replica_count
  tikv_count = var.tikv_replica_count
  tidb_count = var.tidb_replica_count
  tidb_cluster_chart_version = var.tidb_cluster_chart_version
  override_values = var.override_values
  kubeconfig_filename = var.kubeconfig_path
}

resource "null_resource" "wait-lb-ip" {
  provisioner "local-exec" {
    interpreter = ["bash", "-c"]
    working_dir = path.cwd
    command         = <<EOS
set -euo pipefail

until kubectl get svc -n ${var.cluster_name} tidb-cluster-tidb -o json | jq '.status.loadBalancer.ingress[0]' | grep ip; do
  echo "Wait for TiDB internal loadbalancer IP"
  sleep 5
done
EOS

    environment = {
      KUBECONFIG = var.kubeconfig_path
    }
  }
}