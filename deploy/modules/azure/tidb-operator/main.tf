resource "null_resource" "setup-operator" {
  provisioner "local-exec" {
    command = "kubectl apply -f https://raw.githubusercontent.com/pingcap/tidb-operator/${var.tidb_operator_version}/manifests/crd.yaml"
    environment = {
      KUBECONFIG = var.kubeconfig_path
    }
  }
}

resource "helm_release" "tidb-operator" {
  depends_on         = [null_resource.setup-operator]

  name               = "tidb-operator"
  repository         = "https://charts.pingcap.org/"
  chart              = "tidb-operator"
  version            = var.tidb_operator_version
  namespace          = "tidb-admin"
  create_namespace   = true
  values     = [var.operator_helm_values]
}
