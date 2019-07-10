resource "local_file" "kubeconfig" {
  depends_on = [alicloud_cs_managed_kubernetes.k8s]
  sensitive_content = file(alicloud_cs_managed_kubernetes.k8s.kube_config)
  filename = alicloud_cs_managed_kubernetes.k8s.kube_config
}

provider "helm" {
  alias = "initial"
  insecure = true
  install_tiller = false
  kubernetes {
    config_path = local_file.kubeconfig.filename
  }
}

resource "null_resource" "setup-env" {
  depends_on = [local_file.kubeconfig]
  working_dir = path.cwd
  connection = <<EOS
kubectl apply -f https://raw.githubusercontent.com/pingcap/tidb-operator/v1.0.0-beta.3/manifests/crd.yaml
kubeclt apply -f
helm init
until helm ls; do
  echo "Wait tiller ready"
done
EOS
  environment = {
    KUBECONFIG = alicloud_cs_managed_kubernetes.k8s.kube_config
  }
}

data "helm_repository" "pingcap" {
  provider = "helm.initial"
  depends_on = ["null_resource.setup-env"]
  name = "pingcap"
  url = "http://charts.pingcap.org/"
}

resource "helm_release" "tidb-operator" {
  provider = "helm.initial"
  depends_on = ["null_resource.setup-env"]

  repository = data.helm_repository.pingcap.name
  chart = "tidb-operator"
  version = var.operator_version
  namespace = "tidb-admin"
  name = "tidb-operator"
  values = [var.operator_helm_values]
}