data "template_file" "kubeconfig_filename" {
  template = var.kubeconfig_file
  vars = {
    kubernetes_depedency = alicloud_cs_managed_kubernetes.k8s.client_cert
  }
}
resource "null_resource" "setup-env" {
  depends_on = [data.template_file.kubeconfig_filename]

  provisioner "local-exec" {
    working_dir = path.cwd
    command     = <<EOS
mkdir monitor
echo "${local_file.kubeconfig.sensitive_content}" > monitor/config.yaml
# Kubernetes cluster monitor
if ${var.install_prometheus_operator}; then
    wget https://raw.githubusercontent.com/pingcap/monitoring/master/k8s-cluster-monitor/manifests/archive/prometheus-operator.tar.gz
    tar -zxvf prometheus-operator.tar.gz -C monitor/
    kubectl apply -f monitor/manifests/prometheus-operator
fi

if ${var.install_kubernetes_monitor}; then
    wget https://raw.githubusercontent.com/pingcap/monitoring/master/k8s-cluster-monitor/manifests/archive/prometheus.tar.gz
    tar -zxvf prometheus.tar.gz -C monitor/
    sed -i'.bak' 's/local-storage/alicloud-disk-available/g' monitor/manifests/prometheus/grafana-pvc.yaml
    sed -i'.bak' 's/local-storage/alicloud-disk-available/g' monitor/manifests/prometheus/prometheus-prometheus.yaml
    rm -rf monitor/manifests/prometheus/*.bak
    kubectl apply -f monitor/manifests/prometheus
fi

rm -rf prometheus*.tar.gz
rm -rf monitor/
EOS
    environment = {
      KUBECONFIG = "data.template_file.kubeconfig_filename.rendered"
    }
  }
}