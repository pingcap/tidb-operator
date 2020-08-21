# Hack, instruct terraform that the kubeconfig_filename is only available until the k8s created
data "template_file" "kubeconfig_filename" {
  template = var.kubeconfig_file
  vars = {
    kubernetes_dependency = alicloud_cs_managed_kubernetes.k8s.client_cert
  }
}

provider "helm" {
  alias          = "initial"
  insecure       = true
  install_tiller = false
  kubernetes {
    config_path = data.template_file.kubeconfig_filename.rendered
  }
}


resource "null_resource" "setup-env" {
  depends_on = [data.template_file.kubeconfig_filename]

  provisioner "local-exec" {
    working_dir = path.cwd
    # Note for the patch command: ACK has a toleration issue with the pre-deployed flexvolume daemonset, we have to patch
    # it manually and the resource namespace & name are hard-coded by convention
    command = <<EOS
kubectl apply -f https://raw.githubusercontent.com/pingcap/tidb-operator/${var.operator_version}/manifests/crd.yaml
kubectl apply -f https://raw.githubusercontent.com/pingcap/tidb-operator/${var.operator_version}/manifests/tiller-rbac.yaml
kubectl apply -f ${path.module}/manifest/alicloud-disk-storageclass.yaml
echo '${data.template_file.local-volume-provisioner.rendered}' | kubectl apply -f -
kubectl patch -n kube-system daemonset flexvolume --type='json' -p='[{"op":"replace", "path": "/spec/template/spec/tolerations", "value":[{"operator": "Exists"}]}]'
helm init --upgrade --tiller-image registry.cn-hangzhou.aliyuncs.com/google_containers/tiller:$(helm version --client --short | grep -Eo 'v[0-9]\.[0-9]+\.[0-9]+') --service-account tiller
until helm ls; do
  echo "Wait tiller ready"
done
EOS
    environment = {
      KUBECONFIG = data.template_file.kubeconfig_filename.rendered
    }
  }
}

data "helm_repository" "pingcap" {
  provider = helm.initial
  depends_on = [null_resource.setup-env]
  name = "pingcap"
  url = "http://charts.pingcap.org/"
}

resource "helm_release" "tidb-operator" {
  provider = helm.initial
  depends_on = [null_resource.setup-env]

  repository = data.helm_repository.pingcap.name
  chart = "tidb-operator"
  version = var.operator_version
  namespace = "tidb-admin"
  name = "tidb-operator"
  values = [var.operator_helm_values]

  set {
    name = "scheduler.kubeSchedulerImageName"
    value = "registry.cn-hangzhou.aliyuncs.com/google_containers/kube-scheduler-amd64"
  }
}
