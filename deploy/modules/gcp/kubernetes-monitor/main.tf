resource "null_resource" "setup-env" {
  depends_on = [
    var.install_prometheus_operator,
    var.install_kubernetes_monitor,
  ]
  provisioner "local-exec" {
    working_dir = path.cwd
    interpreter = ["bash", "-c"]
    command     = <<EOS
# Kubernetes cluster monitor
mkdir monitor
if var.install_prometheus_operator; then
    wget https://raw.githubusercontent.com/pingcap/monitoring/master/k8s-cluster-monitor/manifests/archive/prometheus-operator.tar.gz
    tar -zxvf prometheus-operator.tar.gz -C monitor/
    kubectl apply -f monitor/manifests/prometheus-operator
fi

if var.install_kubernetes_monitor; then
    wget https://raw.githubusercontent.com/pingcap/monitoring/master/k8s-cluster-monitor/manifests/archive/prometheus.tar.gz
    tar -zxvf prometheus.tar.gz -C monitor/
    sed -i'.bak' 's/local-storage/ebs-gp2/g' monitor/manifests/prometheus/grafana-pvc.yaml
    sed -i'.bak' 's/local-storage/ebs-gp2/g' monitor/manifests/prometheus/prometheus-prometheus.yaml
    rm -rf monitor/manifests/prometheus/*.bak
    kubectl apply -f monitor/manifests/prometheus
fi

rm -rf prometheus*.tar.gz
rm -rf monitor/
EOS
    environment = {
      KUBECONFIG = var.kubeconfig_path
    }
  }
}